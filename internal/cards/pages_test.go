package cards

import "testing"

// The page counts are a promise made before anyone spends paper, so they are
// pinned to the numbers the design states rather than left to drift.
func TestPageCounts(t *testing.T) {
	tests := []struct {
		name       string
		printings  int
		wantSheets int
	}{
		{"empty list costs no paper", 0, 0},
		{"a single printing still needs a page", 1, 1},
		{"a full page", 9, 1},
		{"one over a page", 10, 2},
		{"Charizard with variants", 214, 24},
		{"Charizard without variants", 71, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SheetPages(tt.printings); got != tt.wantSheets {
				t.Errorf("SheetPages(%d) = %d, want %d", tt.printings, got, tt.wantSheets)
			}
		})
	}
}

// Turning variants off must read as "folded away", never as "cards deleted", so the
// result reports what the toggle is currently hiding.
func TestResultFolding(t *testing.T) {
	on := Result{PrintingCount: 214, VariantCount: 214}
	if on.HasFoldedPrintings() {
		t.Error("with variants on, nothing is folded")
	}
	if on.FoldedPrintings() != 0 || on.FoldedPages() != 0 {
		t.Errorf("folded = %d cards / %d pages, want 0/0", on.FoldedPrintings(), on.FoldedPages())
	}

	off := Result{PrintingCount: 71, VariantCount: 214}
	if !off.HasFoldedPrintings() {
		t.Fatal("with variants off, 143 printings are folded away")
	}
	if got := off.FoldedPrintings(); got != 143 {
		t.Errorf("FoldedPrintings() = %d, want 143", got)
	}
	if got := off.FoldedPages(); got != 16 {
		t.Errorf("FoldedPages() = %d, want 16", got)
	}
	if got := off.SheetPages(); got != 8 {
		t.Errorf("SheetPages() = %d, want 8", got)
	}
}

func TestSpeciesSlug(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Charizard", "charizard"},
		{"Mr. Mime", "mr-mime"},
		{"Mime Jr.", "mime-jr"},
		{"Farfetch'd", "farfetchd"},
		{"Nidoran♀", "nidoran-f"},
		{"Nidoran♂", "nidoran-m"},
		{"Ho-Oh", "ho-oh"},
		{"Porygon-Z", "porygon-z"},
		{"Type: Null", "type-null"},
		{"Flabébé", "flabebe"},
		{"Tapu Koko", "tapu-koko"},
	}
	for _, tt := range tests {
		if got := (Species{Name: tt.name}).Slug(); got != tt.want {
			t.Errorf("Species{%q}.Slug() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// A shared URL has to survive being retyped by hand, so every spelling of a name
// resolves to the same Pokémon.
func TestSpeciesIndexByName(t *testing.T) {
	idx := NewSpeciesIndex([]Species{
		{DexID: 6, Name: "Charizard"},
		{DexID: 29, Name: "Nidoran♀"},
		{DexID: 122, Name: "Mr. Mime"},
		{DexID: 669, Name: "Flabébé"},
	})

	tests := []struct {
		query string
		want  int
	}{
		{"charizard", 6},
		{"Charizard", 6},
		{"CHARIZARD", 6},
		{"mr-mime", 122},
		{"mrmime", 122},
		{"Mr. Mime", 122},
		{"nidoran-f", 29},
		{"nidoran♀", 29},
		{"flabebe", 669},
		{"Flabébé", 669},
	}
	for _, tt := range tests {
		sp, ok := idx.ByName(tt.query)
		if !ok {
			t.Errorf("ByName(%q) found nothing, want #%d", tt.query, tt.want)
			continue
		}
		if sp.DexID != tt.want {
			t.Errorf("ByName(%q) = #%d, want #%d", tt.query, sp.DexID, tt.want)
		}
	}

	if _, ok := idx.ByName("notapokemon"); ok {
		t.Error("ByName found a Pokémon that does not exist")
	}

	// Every Pokémon's own slug must round-trip, or its shareable URL is broken.
	for _, sp := range []Species{{DexID: 29, Name: "Nidoran♀"}, {DexID: 122, Name: "Mr. Mime"}} {
		got, ok := idx.ByName(sp.Slug())
		if !ok || got.DexID != sp.DexID {
			t.Errorf("slug %q does not resolve back to #%d", sp.Slug(), sp.DexID)
		}
	}
}

// A missing field must read as "unknown" everywhere, because a blank cell reads as
// a bug. See docs/variant-taxonomy.md.
func TestPrintingDisplay(t *testing.T) {
	full := Printing{
		Name: "Dark Charizard", SetName: "Team Rocket", Number: "4", SetTotal: 82,
		Rarity: "Holo Rare", Illustrator: "Ken Sugimori", Language: LangEN,
	}
	full.ReleaseDate, _ = ParseDate("2000-04-24")

	for field, want := range map[string]string{
		"name":        "Dark Charizard",
		"set":         "Team Rocket",
		"number":      "4/82",
		"rarity":      "Holo Rare",
		"illustrator": "Ken Sugimori",
		"language":    "English",
		"released":    "2000-04-24",
		"year":        "2000",
	} {
		if got := full.Display(field); got != want {
			t.Errorf("Display(%q) = %q, want %q", field, got, want)
		}
	}

	bare := Printing{Language: LangJA}
	for _, field := range []string{"name", "set", "number", "rarity", "illustrator", "released", "year"} {
		if got := bare.Display(field); got != "unknown" {
			t.Errorf("Display(%q) on an empty printing = %q, want %q", field, got, "unknown")
		}
	}
}
