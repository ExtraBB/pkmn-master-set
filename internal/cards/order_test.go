package cards

import (
	"slices"
	"testing"
)

// Plain string comparison puts card 10 before card 9, which would shuffle a
// printed sheet out of the order printed on the cards themselves.
func TestCompareCardNumberSortsNaturally(t *testing.T) {
	tests := []struct {
		in   []string
		want []string
	}{
		{[]string{"10", "9", "1", "100", "2"}, []string{"1", "2", "9", "10", "100"}},
		{[]string{"4a", "4", "5"}, []string{"4", "4a", "5"}},
		{[]string{"TG12", "TG05", "TG2"}, []string{"TG2", "TG05", "TG12"}},
		{[]string{"SV107", "SV9", "H12", "H2"}, []string{"H2", "H12", "SV9", "SV107"}},
		// Zero-padded and bare forms of the same number both appear in the data
		// (Japanese sets pad, English ones mostly don't). They compare equal
		// numerically, so the literal text breaks the tie — which form comes
		// first doesn't matter, only that the order is total and repeatable.
		{[]string{"021", "21", "3"}, []string{"3", "021", "21"}},
		// A number sorts before a letter at the same position.
		{[]string{"XY01", "12"}, []string{"12", "XY01"}},
	}

	for _, tc := range tests {
		got := slices.Clone(tc.in)
		slices.SortFunc(got, CompareCardNumber)
		if !slices.Equal(got, tc.want) {
			t.Errorf("sort(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func mustDate(t *testing.T, s string) Date {
	t.Helper()
	d, err := ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", s, err)
	}
	return d
}

// The binder should read as the Pokémon's history front to back.
func TestSortPrintingsIsChronological(t *testing.T) {
	ps := []Printing{
		{CardID: "neo1-66", SetID: "neo1", SetName: "Neo Genesis", Number: "66", ReleaseDate: mustDate(t, "2000-12-16")},
		{CardID: "base1-4", SetID: "base1", SetName: "Base Set", Number: "4", ReleaseDate: mustDate(t, "1999-01-09")},
		{CardID: "base1-10", SetID: "base1", SetName: "Base Set", Number: "10", ReleaseDate: mustDate(t, "1999-01-09")},
		{CardID: "gym2-2", SetID: "gym2", SetName: "Gym Challenge", Number: "2", ReleaseDate: mustDate(t, "2000-10-16")},
	}

	SortPrintings(ps)

	want := []string{"base1-4", "base1-10", "gym2-2", "neo1-66"}
	got := make([]string, len(ps))
	for i, p := range ps {
		got[i] = p.CardID
	}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// A dateless promo set at the front of the binder would corrupt the chronology,
// so unknown dates go to the end.
func TestUndatedSetsSortLast(t *testing.T) {
	ps := []Printing{
		{CardID: "2024sv-1", SetID: "2024sv", SetName: "McDonald's Collection 2024", Number: "1"},
		{CardID: "base1-4", SetID: "base1", SetName: "Base Set", Number: "4", ReleaseDate: mustDate(t, "1999-01-09")},
	}

	SortPrintings(ps)

	if ps[0].CardID != "base1-4" || ps[1].CardID != "2024sv-1" {
		t.Errorf("order = %s, %s; want the undated set last", ps[0].CardID, ps[1].CardID)
	}
}

// Two sets released on the same day still need one definite order, or the
// preview and the printed sheet could disagree.
func TestSameDayReleasesHaveAStableOrder(t *testing.T) {
	date := mustDate(t, "2000-02-24")
	ps := []Printing{
		{CardID: "b-1", SetID: "b", SetName: "Team Rocket", Number: "1", ReleaseDate: date},
		{CardID: "a-1", SetID: "a", SetName: "Base Set 2", Number: "1", ReleaseDate: date},
	}

	SortPrintings(ps)

	if ps[0].SetName != "Base Set 2" {
		t.Errorf("first = %q, want Base Set 2", ps[0].SetName)
	}
}

// Printings of one card must stay in print-run order within the card.
func TestPrintingsOfOneCardStayInVariantOrder(t *testing.T) {
	date := mustDate(t, "1999-01-09")
	base := Printing{CardID: "base1-4", SetID: "base1", SetName: "Base Set", Number: "4", ReleaseDate: date}

	ps := []Printing{
		{Variant: v("holo", "unlimited", nil, "standard")},
		{Variant: v("holo", "shadowless", []string{"1st-edition"}, "standard")},
		{Variant: v("holo", "shadowless", nil, "standard")},
	}
	for i := range ps {
		p := base
		p.Variant = ps[i].Variant
		ps[i] = p
	}

	SortPrintings(ps)

	want := []string{"1st Edition · Shadowless · Holo", "Shadowless · Holo", "Unlimited · Holo"}
	got := make([]string, len(ps))
	for i, p := range ps {
		got[i] = p.Variant.Label()
	}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestParseDate(t *testing.T) {
	// An empty date is a real answer, not a parse failure: some promo sets have
	// no recorded release date.
	d, err := ParseDate("")
	if err != nil {
		t.Fatalf("ParseDate(\"\"): %v", err)
	}
	if !d.IsZero() || d.String() != "" || d.Year() != 0 {
		t.Errorf("empty date = %+v", d)
	}

	d = mustDate(t, "1999-01-09")
	if d.Year() != 1999 || d.String() != "1999-01-09" {
		t.Errorf("date = %s, year %d", d.String(), d.Year())
	}

	if _, err := ParseDate("January 1999"); err == nil {
		t.Error("want an error for an unparseable date")
	}
}
