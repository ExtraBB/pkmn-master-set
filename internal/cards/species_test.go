package cards

import (
	"slices"
	"testing"
)

// Search must forgive the things people actually type: wrong case, no accents,
// no punctuation, partial names.
func TestFoldForgivesCasingAccentsAndPunctuation(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Charizard", "charizard"},
		{"CHARIZARD", "charizard"},
		{"Flabébé", "flabebe"},
		{"Farfetch'd", "farfetchd"},
		{"Farfetch’d", "farfetchd"},
		{"Mr. Mime", "mrmime"},
		{"Mime Jr.", "mimejr"},
		{"Nidoran♀", "nidoranf"},
		{"Nidoran♂", "nidoranm"},
		{"Type: Null", "typenull"},
		{"Ho-Oh", "hooh"},
		{"Porygon-Z", "porygonz"},
		{"リザードン", "リザードン"},
	}

	for _, tc := range tests {
		if got := Fold(tc.in); got != tc.want {
			t.Errorf("Fold(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSearchPrefersPrefixMatches(t *testing.T) {
	idx := NewSpeciesIndex([]Species{
		{DexID: 4, Name: "Charmander"},
		{DexID: 5, Name: "Charmeleon"},
		{DexID: 6, Name: "Charizard"},
		{DexID: 25, Name: "Pikachu"},
	})

	got := names(idx.Search("char", 10))
	want := []string{"Charmander", "Charmeleon", "Charizard"}
	if !slices.Equal(got, want) {
		t.Errorf("Search(char) = %v, want %v", got, want)
	}
}

// A substring hit is still useful, but shouldn't outrank a name that starts with
// what you typed.
func TestSearchRanksPrefixAbovePartial(t *testing.T) {
	idx := NewSpeciesIndex([]Species{
		{DexID: 26, Name: "Raichu"},
		{DexID: 25, Name: "Pikachu"},
		{DexID: 172, Name: "Chuchu"},
	})

	if got := names(idx.Search("chu", 10)); got[0] != "Chuchu" {
		t.Errorf("Search(chu) = %v, want the prefix match first", got)
	}
}

func TestSearchForgivesAccentsAndPunctuation(t *testing.T) {
	idx := NewSpeciesIndex([]Species{
		{DexID: 669, Name: "Flabébé"},
		{DexID: 83, Name: "Farfetch'd"},
		{DexID: 122, Name: "Mr. Mime"},
		{DexID: 772, Name: "Type: Null"},
	})

	for _, q := range []string{"flabebe", "FLABEBE", "flabébé"} {
		if got := idx.Search(q, 5); len(got) != 1 || got[0].DexID != 669 {
			t.Errorf("Search(%q) = %v, want Flabébé", q, names(got))
		}
	}
	for q, want := range map[string]int{"farfetchd": 83, "mr mime": 122, "mrmime": 122, "type null": 772} {
		if got := idx.Search(q, 5); len(got) != 1 || got[0].DexID != want {
			t.Errorf("Search(%q) = %v, want dex %d", q, names(got), want)
		}
	}
}

// Collectors think in Dex numbers as often as names.
func TestSearchByDexNumber(t *testing.T) {
	idx := NewSpeciesIndex([]Species{{DexID: 6, Name: "Charizard"}})

	if got := idx.Search("6", 5); len(got) != 1 || got[0].Name != "Charizard" {
		t.Errorf("Search(6) = %v", names(got))
	}
	if got := idx.Search("9999", 5); len(got) != 0 {
		t.Errorf("Search(9999) = %v, want nothing", names(got))
	}
}

func TestSearchRespectsLimit(t *testing.T) {
	idx := NewSpeciesIndex([]Species{
		{DexID: 4, Name: "Charmander"},
		{DexID: 5, Name: "Charmeleon"},
		{DexID: 6, Name: "Charizard"},
	})

	if got := idx.Search("char", 2); len(got) != 2 {
		t.Errorf("got %d results, want 2", len(got))
	}
	if got := idx.Search("char", 0); got != nil {
		t.Errorf("got %v, want nothing for a zero limit", names(got))
	}
}

// The embedded index is what makes search work offline and instantly, so its
// integrity is worth asserting directly.
func TestEmbeddedSpeciesIndexIsComplete(t *testing.T) {
	idx, err := EmbeddedSpecies()
	if err != nil {
		t.Fatalf("EmbeddedSpecies: %v", err)
	}
	if idx.Len() < 1025 {
		t.Errorf("index has %d species, want the full National Dex", idx.Len())
	}

	for dex, want := range map[int]string{
		1: "Bulbasaur", 6: "Charizard", 25: "Pikachu", 29: "Nidoran♀",
		83: "Farfetch'd", 122: "Mr. Mime", 133: "Eevee", 150: "Mewtwo",
		197: "Umbreon", 669: "Flabébé", 772: "Type: Null",
	} {
		got, ok := idx.ByDexID(dex)
		if !ok {
			t.Errorf("dex %d missing", dex)
			continue
		}
		if got.Name != want {
			t.Errorf("dex %d = %q, want %q", dex, got.Name, want)
		}
	}
}

func names(ss []Species) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.Name
	}
	return out
}
