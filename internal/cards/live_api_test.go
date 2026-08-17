//go:build live

// These tests hit the real tcgdex API and are excluded from `go test ./...`.
// They are the check that the product's completeness promise still holds against
// live data, rather than against fixtures we wrote ourselves.
//
//	go test -tags=live ./internal/cards/
//	go test -tags=live -run TestLiveDump -v ./internal/cards/   # print a list
package cards

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ExtraBB/pkmn-master-set/internal/tcgdex"
)

func liveCatalog(t *testing.T) *Catalog {
	t.Helper()
	species, err := EmbeddedSpecies()
	if err != nil {
		t.Fatalf("EmbeddedSpecies: %v", err)
	}
	src := NewLiveSource(tcgdex.New(), time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := src.WarmSets(ctx, LangEN); err != nil {
		t.Fatalf("WarmSets: %v", err)
	}
	return NewCatalog(src, species)
}

// Floors rather than exact counts: new sets ship regularly and should never turn
// this into a failing test, but a drop below what we have already seen means the
// list quietly lost printings.
func TestLiveCardCountFloors(t *testing.T) {
	cat := liveCatalog(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	tests := []struct {
		dex       int
		name      string
		lang      Language
		minCards  int
		minPrints int
	}{
		// Measured 2026-08-17, with a few per cent of headroom so ordinary
		// source churn never turns this red — only a real loss does.
		{6, "Charizard", LangEN, 124, 145},
		{25, "Pikachu", LangEN, 206, 280},
		{133, "Eevee", LangEN, 105, 138},
		{150, "Mewtwo", LangEN, 92, 112},
		{197, "Umbreon", LangEN, 52, 65},
		{6, "Charizard", LangJA, 25, 32},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s_%s", tc.name, tc.lang), func(t *testing.T) {
			got, err := cat.Query(ctx, Query{DexID: tc.dex, Language: tc.lang, IncludeVariants: true})
			if err != nil {
				t.Fatalf("Query: %v", err)
			}
			t.Logf("%s (%s): %d cards, %d printings", tc.name, tc.lang, got.CardCount, got.PrintingCount)
			if got.CardCount < tc.minCards {
				t.Errorf("CardCount = %d, want at least %d", got.CardCount, tc.minCards)
			}
			if got.PrintingCount < tc.minPrints {
				t.Errorf("PrintingCount = %d, want at least %d", got.PrintingCount, tc.minPrints)
			}
		})
	}
}

// Every promised field must be either present or explicitly reported missing —
// never silently blank.
func TestLiveEveryPrintingIsWellFormed(t *testing.T) {
	cat := liveCatalog(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	got, err := cat.Query(ctx, Query{DexID: 6, Language: LangEN, IncludeVariants: true})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	seen := map[string]bool{}
	for _, p := range got.Printings {
		if p.Name == "" || p.Number == "" || p.SetName == "" {
			t.Errorf("printing %q is missing core identity: %+v", p.ID, p)
		}
		if p.VariantLabel == "" {
			t.Errorf("printing %q has no variant label", p.ID)
		}
		if seen[p.ID] {
			t.Errorf("duplicate printing id %q — a collector would cut the same placeholder twice", p.ID)
		}
		seen[p.ID] = true

		if p.Rarity == "" && !p.Lacks("rarity") {
			t.Errorf("printing %q has a blank rarity that is not reported as missing", p.ID)
		}
	}
}

// A card featuring two Pokémon must show up in both collections.
func TestLiveDualPokemonCardAppearsInBothLists(t *testing.T) {
	cat := liveCatalog(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for _, dex := range []int{25, 644} {
		got, err := cat.Query(ctx, Query{DexID: dex, Language: LangEN, IncludeVariants: true})
		if err != nil {
			t.Fatalf("Query(%d): %v", dex, err)
		}
		found := false
		for _, p := range got.Printings {
			if p.CardID == "sm9-33" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Pikachu & Zekrom GX (sm9-33) missing from dex %d's list", dex)
		}
	}
}

// TestLiveChecklist prints one compact line per card for the Pokémon used in the
// completeness fixtures, in the format those fixtures use. It is how the
// reference lists get diffed against reality when they need refreshing.
//
//	go test -tags=live -run TestLiveChecklist -v ./internal/cards/
func TestLiveChecklist(t *testing.T) {
	cat := liveCatalog(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for _, sp := range completenessFixtures {
		got, err := cat.Query(ctx, Query{DexID: sp.dexID, Language: LangEN})
		if err != nil {
			t.Fatalf("Query(%s): %v", sp.name, err)
		}
		fmt.Printf("### %s (%d cards)\n", sp.name, got.CardCount)
		for _, p := range got.Printings {
			fmt.Printf("%s %s\t# %s %s\n", p.SetID, p.Number, p.SetName, p.Name)
		}
	}
}

// TestLiveDump prints a real list so the ordering and variant labels can be
// eyeballed against a collector reference. Not an assertion — a viewing tool.
//
//	go test -tags=live -run TestLiveDump -v ./internal/cards/
func TestLiveDump(t *testing.T) {
	cat := liveCatalog(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	got, err := cat.Query(ctx, Query{DexID: 6, Language: LangEN, IncludeVariants: true})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	fmt.Printf("%s · %s · %d cards · %d printings\n", got.Species.Name, got.Language, got.CardCount, got.PrintingCount)
	for _, p := range got.Printings {
		date := p.ReleaseDate.String()
		if date == "" {
			date = "unknown"
		}
		fmt.Printf("%-32s %-10s %-16s %-40s %s\n", p.SetName, p.NumberOfTotal(), p.Rarity, p.VariantLabel, date)
	}
}
