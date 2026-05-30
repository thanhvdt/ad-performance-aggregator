package csvio

import "testing"

func TestParseRow(t *testing.T) {
	cases := []struct {
		name                            string
		line                            string
		wantID                          string
		wantImp, wantClk, wantC, wantCv int64
		wantOK                          bool
	}{
		{"two decimals", "CMP025,2025-04-18,3653,60,64.29,2", "CMP025", 3653, 60, 6429, 2, true},
		{"one decimal", "CMP001,2025-01-01,14000,340,48.20,15", "CMP001", 14000, 340, 4820, 15, true},
		{"one decimal short", "CMP002,2025-01-02,8500,150,31.5,5", "CMP002", 8500, 150, 3150, 5, true},
		{"integer spend", "CMP003,2025-01-01,5000,60,15,3", "CMP003", 5000, 60, 1500, 3, true},
		{"zero conversions", "CMP004,2025-01-03,1000,50,10.00,0", "CMP004", 1000, 50, 1000, 0, true},
		{"too few fields", "this,row,is,malformed", "", 0, 0, 0, 0, false},
		{"non-numeric impressions", "CMP005,2025-01-03,abc,10,5.00,1", "", 0, 0, 0, 0, false},
		{"too many fields", "CMP006,2025,1,2,3.00,4,extra", "", 0, 0, 0, 0, false},
		{"empty numeric field", "CMP007,2025,,10,5.00,1", "", 0, 0, 0, 0, false},
		{"empty line", "", "", 0, 0, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, imp, clk, cents, conv, ok := ParseRow([]byte(tc.line))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if string(id) != tc.wantID || imp != tc.wantImp || clk != tc.wantClk || cents != tc.wantC || conv != tc.wantCv {
				t.Errorf("got id=%q imp=%d clk=%d cents=%d conv=%d; want id=%q imp=%d clk=%d cents=%d conv=%d",
					id, imp, clk, cents, conv, tc.wantID, tc.wantImp, tc.wantClk, tc.wantC, tc.wantCv)
			}
		})
	}
}

var sampleLine = []byte("CMP025,2025-04-18,3653,60,64.29,2")

func BenchmarkParseRow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, _, _, _, _, ok := ParseRow(sampleLine); !ok {
			b.Fatal("parse failed")
		}
	}
}
