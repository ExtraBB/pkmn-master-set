package cards

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// fakeSource serves fixtures, so nothing in this file touches the network.
type fakeSource struct {
	cards map[Language][]Card
	sets  map[string]Set
	err   error
}

func (f *fakeSource) Cards(_ context.Context, lang Language, dexID int) ([]Card, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []Card
	for _, c := range f.cards[lang] {
		if slices.Contains(c.DexIDs, dexID) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeSource) Set(_ context.Context, _ Language, setID string) (Set, error) {
	s, ok := f.sets[setID]
	if !ok {
		return Set{}, errors.New("no such set")
	}
	return s, nil
}

func testCatalog(t *testing.T, src *fakeSource) *Catalog {
	t.Helper()
	return NewCatalog(src, NewSpeciesIndex([]Species{
		{DexID: 6, Name: "Charizard"},
		{DexID: 25, Name: "Pikachu"},
		{DexID: 644, Name: "Zekrom"},
	}))
}

func charizardFixture(t *testing.T) *fakeSource {
	t.Helper()
	return &fakeSource{
		sets: map[string]Set{
			"base1":  {ID: "base1", Name: "Base Set", Series: "Base", ReleaseDate: mustDate(t, "1999-01-09"), CountOfficial: 102},
			"base2":  {ID: "base2", Name: "Base Set 2", Series: "Base", ReleaseDate: mustDate(t, "2000-02-24"), CountOfficial: 130},
			"2024sv": {ID: "2024sv", Name: "McDonald's Collection 2024"},
		},
		cards: map[Language][]Card{
			LangEN: {
				{
					ID: "base1-4", Name: "Charizard", Number: "4", DexIDs: []int{6},
					Rarity: "Rare Holo", Illustrator: "Mitsuhiro Arita",
					ImageBase: "https://assets.tcgdex.net/en/base/base1/4",
					SetID:     "base1", SetName: "Base Set", SetTotal: 102,
					Variants: []Variant{
						{Type: "holo", Subtype: "unlimited", Size: "standard"},
						{Type: "holo", Subtype: "shadowless", Stamps: []string{"1st-edition"}, Size: "standard"},
						{Type: "holo", Subtype: "shadowless", Size: "standard"},
					},
				},
				{
					ID: "base2-4", Name: "Charizard", Number: "4", DexIDs: []int{6},
					Rarity: "Rare Holo", Illustrator: "Mitsuhiro Arita",
					ImageBase: "https://assets.tcgdex.net/en/base/base2/4",
					SetID:     "base2", SetName: "Base Set 2", SetTotal: 130,
					Variants: []Variant{{Type: "holo", Size: "standard"}},
				},
				{
					// Null image, no rarity, no illustrator, undated set — the
					// awkward card that exercises every "unknown" path at once.
					ID: "2024sv-1", Name: "Charizard", Number: "1", DexIDs: []int{6},
					SetID: "2024sv", SetName: "McDonald's Collection 2024", SetTotal: 15,
				},
			},
		},
	}
}

// tcgdex leaves most variants' own pricing at the zero value even when the
// card overall is priced, so a printing must fall back to the card's pricing
// rather than showing "unknown" whenever its own variant entry has none — but
// still prefer the variant's own data on the rare occasion tcgdex has it.
func TestPrintingPricingFallsBackToCardWhenVariantHasNone(t *testing.T) {
	cardPricing := Pricing{Cardmarket: CardmarketPricing{ProductID: 1, Avg: 10}}
	variantPricing := Pricing{Cardmarket: CardmarketPricing{ProductID: 2, Avg: 20}}

	src := &fakeSource{
		sets: map[string]Set{"base1": {ID: "base1", Name: "Base Set", CountOfficial: 102}},
		cards: map[Language][]Card{
			LangEN: {{
				ID: "base1-4", Name: "Charizard", Number: "4", DexIDs: []int{6},
				SetID: "base1", SetName: "Base Set", SetTotal: 102,
				Pricing: cardPricing,
				Variants: []Variant{
					{Type: "holo", Subtype: "unlimited", Size: "standard", Pricing: variantPricing},
					{Type: "holo", Subtype: "shadowless", Size: "standard"},
				},
			}},
		},
	}

	got, err := testCatalog(t, src).Query(context.Background(), Query{DexID: 6, Language: LangEN, IncludeVariants: true})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(got.Printings) != 2 {
		t.Fatalf("got %d printings, want 2", len(got.Printings))
	}
	// Earliest print run first: Shadowless (no variant pricing of its own) sorts
	// before Unlimited (has its own).
	if p := got.Printings[0]; p.Variant.Subtype != "shadowless" || p.Pricing.Cardmarket != cardPricing.Cardmarket {
		t.Errorf("shadowless printing = %+v pricing %+v, want the card's fallback %+v", p.Variant, p.Pricing.Cardmarket, cardPricing.Cardmarket)
	}
	if p := got.Printings[1]; p.Variant.Subtype != "unlimited" || p.Pricing.Cardmarket != variantPricing.Cardmarket {
		t.Errorf("unlimited printing = %+v pricing %+v, want the variant's own %+v", p.Variant, p.Pricing.Cardmarket, variantPricing.Cardmarket)
	}
}

func TestQueryWithVariantsExpandsEveryPrinting(t *testing.T) {
	cat := testCatalog(t, charizardFixture(t))

	got, err := cat.Query(context.Background(), Query{DexID: 6, Language: LangEN, IncludeVariants: true})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if got.CardCount != 3 {
		t.Errorf("CardCount = %d, want 3", got.CardCount)
	}
	if got.PrintingCount != 5 {
		t.Errorf("PrintingCount = %d, want 5 (3 + 1 + 1)", got.PrintingCount)
	}

	want := []string{
		"Base Set 4/102 1st Edition · Shadowless · Holo",
		"Base Set 4/102 Shadowless · Holo",
		"Base Set 4/102 Unlimited · Holo",
		"Base Set 2 4/130 Holo",
		"McDonald's Collection 2024 1/15 Standard",
	}
	if diff := describe(got.Printings); !slices.Equal(diff, want) {
		t.Errorf("printings =\n %v\nwant\n %v", diff, want)
	}
}

// Turning variants off must fold rows, not delete cards — and must say how many
// it folded, so the count stays honest.
func TestQueryWithoutVariantsCollapsesToOneRowPerCard(t *testing.T) {
	cat := testCatalog(t, charizardFixture(t))

	got, err := cat.Query(context.Background(), Query{DexID: 6, Language: LangEN})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if got.PrintingCount != 3 {
		t.Fatalf("PrintingCount = %d, want 3", got.PrintingCount)
	}
	if got.CardCount != 3 {
		t.Errorf("CardCount = %d, want 3", got.CardCount)
	}
	if n := got.Printings[0].Collapsed; n != 3 {
		t.Errorf("Base Set row stands for %d printings, want 3", n)
	}
	// With variants off the row represents the card, so labelling it with one
	// printing's name would be a lie.
	if l := got.Printings[0].VariantLabel; l != "" {
		t.Errorf("VariantLabel = %q, want empty when collapsed", l)
	}
	if id := got.Printings[0].ID; id != "base1-4" {
		t.Errorf("ID = %q, want the bare card id when collapsed", id)
	}
}

// A blank cell reads as a bug; a stated "unknown" reads as a known gap.
func TestMissingFieldsAreNamedRatherThanBlank(t *testing.T) {
	cat := testCatalog(t, charizardFixture(t))

	got, err := cat.Query(context.Background(), Query{DexID: 6, Language: LangEN, IncludeVariants: true})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	// The McDonald's card sorts last because its set has no release date.
	last := got.Printings[len(got.Printings)-1]
	for _, field := range []string{"rarity", "illustrator", "image", "releaseDate"} {
		if !last.Lacks(field) {
			t.Errorf("%q not reported as missing; Missing = %v", field, last.Missing)
		}
	}
	if last.ImageURL("low", "webp") != "" {
		t.Error("want an empty image URL when the source has no image")
	}

	first := got.Printings[0]
	if len(first.Missing) != 0 {
		t.Errorf("complete card reported missing fields: %v", first.Missing)
	}
	if got, want := first.ImageURL("high", "png"), "https://assets.tcgdex.net/en/base/base1/4/high.png"; got != want {
		t.Errorf("ImageURL = %q, want %q", got, want)
	}
}

// A card featuring two Pokémon belongs in both collections.
func TestDualPokemonCardsAppearInBothLists(t *testing.T) {
	src := &fakeSource{
		sets: map[string]Set{"sm9": {ID: "sm9", Name: "Team Up", ReleaseDate: mustDate(t, "2019-02-01"), CountOfficial: 181}},
		cards: map[Language][]Card{LangEN: {{
			ID: "sm9-33", Name: "Pikachu & Zekrom GX", Number: "33", DexIDs: []int{25, 644},
			Rarity: "Ultra Rare", Illustrator: "Mitsuhiro Arita", ImageBase: "x",
			SetID: "sm9", SetName: "Team Up", SetTotal: 181,
		}}},
	}
	cat := testCatalog(t, src)

	for _, dex := range []int{25, 644} {
		got, err := cat.Query(context.Background(), Query{DexID: dex, Language: LangEN, IncludeVariants: true})
		if err != nil {
			t.Fatalf("Query(%d): %v", dex, err)
		}
		if got.PrintingCount != 1 {
			t.Errorf("dex %d: got %d printings, want 1", dex, got.PrintingCount)
		}
	}
}

// A Pokémon with no cards in a language is an honest empty answer, not a failure.
// This is common for Japanese.
func TestNoCardsInLanguageIsEmptyNotAnError(t *testing.T) {
	cat := testCatalog(t, charizardFixture(t))

	got, err := cat.Query(context.Background(), Query{DexID: 6, Language: LangJA, IncludeVariants: true})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if got.PrintingCount != 0 {
		t.Errorf("got %d printings, want 0", got.PrintingCount)
	}
	if got.Species.Name != "Charizard" {
		t.Errorf("Species = %q, want Charizard", got.Species.Name)
	}
}

func TestQueryRejectsUnknownSpeciesAndLanguage(t *testing.T) {
	cat := testCatalog(t, charizardFixture(t))

	if _, err := cat.Query(context.Background(), Query{DexID: 99999, Language: LangEN}); !errors.Is(err, ErrUnknownSpecies) {
		t.Errorf("err = %v, want ErrUnknownSpecies", err)
	}
	if _, err := cat.Query(context.Background(), Query{DexID: 6, Language: "fr"}); !errors.Is(err, ErrUnsupportedLanguage) {
		t.Errorf("err = %v, want ErrUnsupportedLanguage", err)
	}
}

// A set we cannot describe must not drop the card: the collector still needs a
// placeholder for it.
func TestUnresolvableSetStillYieldsPrintings(t *testing.T) {
	src := charizardFixture(t)
	delete(src.sets, "base1")
	cat := testCatalog(t, src)

	got, err := cat.Query(context.Background(), Query{DexID: 6, Language: LangEN, IncludeVariants: true})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if got.PrintingCount != 5 {
		t.Fatalf("PrintingCount = %d, want 5", got.PrintingCount)
	}
	// It falls back to the set name carried on the card, and joins the undated
	// sets at the end for want of a release date.
	var fallback *Printing
	for i, p := range got.Printings {
		if p.CardID == "base1-4" {
			fallback = &got.Printings[i]
			break
		}
	}
	if fallback == nil {
		t.Fatal("base1-4 dropped when its set could not be resolved")
	}
	if fallback.SetName != "Base Set" || !fallback.Lacks("releaseDate") {
		t.Errorf("fallback row = %+v", *fallback)
	}
	// Base Set 2 is the only dated set left, so it must now sort first.
	if got.Printings[0].SetID != "base2" {
		t.Errorf("first row = %q, want the only dated set", got.Printings[0].SetID)
	}
}

// Pokémon TCG Pocket cards exist only in a phone app, so a collector can never
// put one in a binder: they must not appear as rows, and must not inflate the
// counts the preview reports.
func TestQuerySkipsDigitalOnlySeries(t *testing.T) {
	src := charizardFixture(t)
	src.sets["A1"] = Set{ID: "A1", Name: "Genetic Apex", SeriesID: "tcgp", Series: "Pokémon TCG Pocket", ReleaseDate: mustDate(t, "2024-10-30"), CountOfficial: 226}
	src.cards[LangEN] = append(src.cards[LangEN], Card{
		ID: "A1-36", Name: "Charizard", Number: "36", DexIDs: []int{6},
		SetID: "A1", SetName: "Genetic Apex", SetTotal: 226,
		Variants: []Variant{{Type: "holo", Size: "standard"}},
	})

	got, err := testCatalog(t, src).Query(context.Background(), Query{DexID: 6, Language: LangEN, IncludeVariants: true})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	for _, p := range got.Printings {
		if p.SetID == "A1" {
			t.Fatalf("tcgp printing survived: %+v", p)
		}
	}
	// The three fixture cards and their five printings, with the Pocket card
	// counted nowhere.
	if got.CardCount != 3 || got.PrintingCount != 5 || got.VariantCount != 5 {
		t.Errorf("counts = %d cards / %d printings / %d variants, want 3/5/5", got.CardCount, got.PrintingCount, got.VariantCount)
	}
}

func describe(ps []Printing) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.SetName + " " + p.NumberOfTotal() + " " + p.Variant.Label()
	}
	return out
}
