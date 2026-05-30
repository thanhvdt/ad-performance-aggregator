package aggregate

import (
	"cmp"
	"slices"
	"strings"
)

// TopByCTR returns the n campaigns with the highest CTR, ties broken by
// campaign_id for deterministic output. The input slice is not modified.
func TopByCTR(cs []Campaign, n int) []Campaign {
	ranked := slices.Clone(cs)
	slices.SortFunc(ranked, func(a, b Campaign) int {
		return cmp.Or(cmp.Compare(b.CTR, a.CTR), strings.Compare(a.ID, b.ID))
	})
	return head(ranked, n)
}

// TopByCPA returns the n campaigns with the lowest CPA, excluding campaigns with
// zero conversions, ties broken by campaign_id for deterministic output.
func TopByCPA(cs []Campaign, n int) []Campaign {
	ranked := make([]Campaign, 0, len(cs))
	for _, c := range cs {
		if c.HasCPA {
			ranked = append(ranked, c)
		}
	}
	slices.SortFunc(ranked, func(a, b Campaign) int {
		return cmp.Or(cmp.Compare(a.CPA, b.CPA), strings.Compare(a.ID, b.ID))
	})
	return head(ranked, n)
}

// head returns the first n elements, or all of them when there are fewer.
func head(cs []Campaign, n int) []Campaign {
	if n >= 0 && len(cs) > n {
		return cs[:n]
	}
	return cs
}
