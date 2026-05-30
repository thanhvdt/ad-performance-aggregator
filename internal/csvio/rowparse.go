// Package csvio reads and writes this challenge's advertising-performance CSV
// format, separate from the aggregation domain (internal/aggregate). It owns all
// knowledge of the column layout: ParseRow reads an input row, and
// WriteRankings/WriteCSV emit the result files — input and output schema in one
// place.
//
// ParseRow is a zero-allocation, byte-level parse of a single data row. It
// assumes the dataset's measured shape: exactly 6 comma-separated fields, no
// quoting, non-negative integer columns, and a spend with at most two decimal
// digits. Anything that doesn't fit fails the parse so the caller can skip the
// row. The returned campaignID slice aliases the input and is only valid until
// the input buffer is reused.
package csvio

// ParseRow extracts the campaign_id and the four numeric columns from one data
// row (campaign_id, date, impressions, clicks, spend, conversions) in a single
// left-to-right pass, with no allocations. ok is false unless the row is exactly
// six fields and every number parses. Spend is returned as integer cents.
func ParseRow(line []byte) (campaignID []byte, impressions, clicks, cents, conversions int64, ok bool) {
	rest := line
	if campaignID, rest, ok = field(rest); !ok { // campaign_id
		return
	}
	if _, rest, ok = field(rest); !ok { // date — not needed for aggregation
		return
	}
	if impressions, rest, ok = uintField(rest); !ok {
		return
	}
	if clicks, rest, ok = uintField(rest); !ok {
		return
	}
	if cents, rest, ok = centsField(rest); !ok {
		return
	}
	conversions, ok = lastUint(rest) // final field: digits only, no trailing comma
	return
}

// field splits off the next comma-delimited field, returning it and the bytes
// after the comma. ok is false when there is no comma (too few fields).
func field(b []byte) (val, rest []byte, ok bool) {
	for i := 0; i < len(b); i++ {
		if b[i] == ',' {
			return b[:i], b[i+1:], true
		}
	}
	return nil, nil, false
}

// uintField reads a non-negative integer field terminated by a comma, returning
// the value and the bytes after the comma.
func uintField(b []byte) (v int64, rest []byte, ok bool) {
	var i int
	for ; i < len(b) && b[i] != ','; i++ {
		if b[i] < '0' || b[i] > '9' {
			return 0, nil, false
		}
		v = v*10 + int64(b[i]-'0')
	}
	if i == 0 || i == len(b) {
		return 0, nil, false // empty field, or no terminating comma
	}
	return v, b[i+1:], true
}

// lastUint reads the final integer field: digits running to the end of the line.
// A comma here would be a non-digit, which correctly rejects an extra field.
func lastUint(b []byte) (v int64, ok bool) {
	if len(b) == 0 {
		return 0, false
	}
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}
		v = v*10 + int64(c-'0')
	}
	return v, true
}

// centsField reads a spend field (integer part plus one or two decimal digits)
// terminated by a comma, returning the value in integer cents.
func centsField(b []byte) (cents int64, rest []byte, ok bool) {
	var dollars, frac int64
	var fracDigits int
	var seenDot, anyDigit bool
	var i int
	for ; i < len(b) && b[i] != ','; i++ {
		switch c := b[i]; {
		case c == '.':
			if seenDot {
				return 0, nil, false
			}
			seenDot = true
		case c >= '0' && c <= '9':
			anyDigit = true
			switch {
			case !seenDot:
				dollars = dollars*10 + int64(c-'0')
			case fracDigits < 2:
				frac = frac*10 + int64(c-'0')
				fracDigits++
			}
		default:
			return 0, nil, false
		}
	}
	if !anyDigit || i == len(b) {
		return 0, nil, false // no digits, or no terminating comma
	}
	if fracDigits == 1 {
		frac *= 10
	}
	return dollars*100 + frac, b[i+1:], true
}
