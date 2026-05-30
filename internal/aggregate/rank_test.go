package aggregate

import "testing"

func TestTopByCTR(t *testing.T) {
	cs := []Campaign{
		{ID: "A", CTR: 0.01}, {ID: "B", CTR: 0.05},
		{ID: "C", CTR: 0.03}, {ID: "D", CTR: 0.03}, // tie -> id order
	}
	assertOrder(t, TopByCTR(cs, 3), []string{"B", "C", "D"}) // desc CTR, ties by id
	if len(TopByCTR(cs, 10)) != 4 {
		t.Errorf("n larger than input should return all")
	}
}

func TestTopByCPAExcludesZeroConversions(t *testing.T) {
	cs := []Campaign{
		{ID: "A", CPA: 20, HasCPA: true},
		{ID: "B", CPA: 10, HasCPA: true},
		{ID: "Z", HasCPA: false}, // zero conversions -> excluded
		{ID: "C", CPA: 15, HasCPA: true},
	}
	got := TopByCPA(cs, 10)
	assertOrder(t, got, []string{"B", "C", "A"}) // asc CPA, Z dropped
	for _, c := range got {
		if c.ID == "Z" {
			t.Fatal("zero-conversion campaign must be excluded from CPA ranking")
		}
	}
}

func assertOrder(t *testing.T, got []Campaign, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), want)
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Errorf("position %d = %s, want %s", i, got[i].ID, id)
		}
	}
}
