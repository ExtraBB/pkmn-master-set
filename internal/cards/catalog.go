package cards

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrUnknownSpecies means the requested National Dex number is not a Pokémon
	// we know about — a bad URL, not an empty result.
	ErrUnknownSpecies = errors.New("cards: unknown species")
	// ErrUnsupportedLanguage means a language outside EN/JA was requested.
	ErrUnsupportedLanguage = errors.New("cards: unsupported language")
)

// Source supplies raw card and set data. The live implementation talks to
// tcgdex; tests supply fixtures, so nothing below this line needs a network.
type Source interface {
	// Cards returns every card in lang whose dex ID list contains dexID.
	Cards(ctx context.Context, lang Language, dexID int) ([]Card, error)
	// Set returns set metadata, most importantly the release date that the card
	// responses omit.
	Set(ctx context.Context, lang Language, setID string) (Set, error)
}

// Catalog answers product questions about a Pokémon's card list.
type Catalog struct {
	src     Source
	species *SpeciesIndex
}

func NewCatalog(src Source, species *SpeciesIndex) *Catalog {
	return &Catalog{src: src, species: species}
}

// Query asks for one Pokémon's list in one language.
type Query struct {
	DexID    int
	Language Language
	// IncludeVariants controls whether each distinct printing gets its own row
	// (and its own placeholder to print) or the card is collapsed to one row.
	// This is the option that moves the page count most.
	IncludeVariants bool
}

// Result is the full answer, in the order every output renders it.
type Result struct {
	Species         Species    `json:"species"`
	Language        Language   `json:"language"`
	IncludeVariants bool       `json:"includeVariants"`
	Printings       []Printing `json:"printings"`
	// CardCount is distinct physical cards; PrintingCount is rows, which is what
	// determines how much paper this will take.
	CardCount     int `json:"cardCount"`
	PrintingCount int `json:"printingCount"`
	// VariantCount is how many printings exist regardless of the toggle. With
	// variants on it equals PrintingCount; with variants off it is larger, and the
	// difference is what the preview reports so turning variants off reads as
	// "folded away", never as "deleted".
	VariantCount int       `json:"variantCount"`
	FetchedAt    time.Time `json:"fetchedAt"`
}

// Search finds Pokémon by name or dex number for the typeahead.
func (c *Catalog) Search(q string, limit int) []Species {
	return c.species.Search(q, limit)
}

// Species looks up one Pokémon by National Dex number.
func (c *Catalog) Species(dexID int) (Species, bool) { return c.species.ByDexID(dexID) }

// SpeciesByName resolves a name or URL slug to one Pokémon.
func (c *Catalog) SpeciesByName(q string) (Species, bool) { return c.species.ByName(q) }

// Query returns the complete, ordered card list for one Pokémon.
//
// A known Pokémon with no cards in the chosen language is not an error — it is an
// empty result, so the UI can say "no Japanese cards" honestly instead of showing
// a failure.
func (c *Catalog) Query(ctx context.Context, q Query) (Result, error) {
	if !q.Language.Valid() {
		return Result{}, fmt.Errorf("%w: %q", ErrUnsupportedLanguage, q.Language)
	}
	sp, ok := c.species.ByDexID(q.DexID)
	if !ok {
		return Result{}, fmt.Errorf("%w: %d", ErrUnknownSpecies, q.DexID)
	}

	raw, err := c.src.Cards(ctx, q.Language, q.DexID)
	if err != nil {
		return Result{}, err
	}

	sets, err := c.resolveSets(ctx, q.Language, raw)
	if err != nil {
		return Result{}, err
	}

	printings := make([]Printing, 0, len(raw))
	variantCount := 0
	for _, card := range raw {
		rows := expand(card, sets[card.SetID], q.Language, q.IncludeVariants)
		for _, p := range rows {
			variantCount += p.Collapsed
		}
		printings = append(printings, rows...)
	}
	SortPrintings(printings)

	return Result{
		Species:         sp,
		Language:        q.Language,
		IncludeVariants: q.IncludeVariants,
		Printings:       printings,
		CardCount:       len(raw),
		PrintingCount:   len(printings),
		VariantCount:    variantCount,
		FetchedAt:       time.Now(),
	}, nil
}

// resolveSets fetches metadata for every set the cards belong to.
//
// A set the source cannot describe is not fatal: the card still exists and a
// collector still needs a placeholder for it. It gets an unknown release date and
// sorts to the end, with the gap recorded on each printing.
func (c *Catalog) resolveSets(ctx context.Context, lang Language, raw []Card) (map[string]Set, error) {
	sets := make(map[string]Set)
	for _, card := range raw {
		if card.SetID == "" {
			continue
		}
		if _, done := sets[card.SetID]; done {
			continue
		}
		s, err := c.src.Set(ctx, lang, card.SetID)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			s = Set{ID: card.SetID, Name: card.SetName}
		}
		sets[card.SetID] = s
	}
	return sets, nil
}

// expand turns one card into its output rows: one per printing with variants on,
// or a single collapsed row with variants off.
func expand(card Card, set Set, lang Language, includeVariants bool) []Printing {
	variants := NormalizeVariants(card.Variants)

	base := Printing{
		CardID:      card.ID,
		Name:        card.Name,
		SetID:       card.SetID,
		SetName:     preferName(set.Name, card.SetName),
		Series:      set.Series,
		Number:      card.Number,
		SetTotal:    preferCount(set.CountOfficial, card.SetTotal),
		Rarity:      card.Rarity,
		Language:    lang,
		ReleaseDate: set.ReleaseDate,
		Illustrator: card.Illustrator,
		ImageBase:   card.ImageBase,
	}
	base.Missing = missingFields(base)

	if !includeVariants {
		p := base
		p.Variant = Collapse(variants)
		p.Pricing = resolvePricing(p.Variant, card)
		p.ID = card.ID
		p.Collapsed = len(variants)
		// The label is left empty: with variants off the row stands for the card
		// as a whole, and captioning it with one printing's name would be a lie.
		return []Printing{p}
	}

	out := make([]Printing, 0, len(variants))
	for _, v := range variants {
		p := base
		p.Variant = v
		p.Pricing = resolvePricing(v, card)
		p.ID = card.ID + "#" + v.Key()
		p.VariantLabel = v.Label()
		p.VariantShort = v.Short()
		p.Collapsed = 1
		out = append(out, p)
	}
	return out
}

// resolvePricing picks the pricing block a printing renders. tcgdex populates a
// variant's own pricing for only a minority of variants — most carry the zero
// value even when the card overall is priced — so the card's pricing, which
// tcgdex fills in far more reliably and which carries every finish it has a
// listing for, is the fallback rather than the exception.
func resolvePricing(v Variant, card Card) Pricing {
	if !v.Pricing.IsZero() {
		return v.Pricing
	}
	return card.Pricing
}

// missingFields names every promised field the source had no value for, so output
// can show "unknown" instead of an ambiguous blank.
func missingFields(p Printing) []string {
	var missing []string
	for _, f := range []struct {
		name  string
		empty bool
	}{
		{"name", p.Name == ""},
		{"set", p.SetName == ""},
		{"number", p.Number == ""},
		{"setTotal", p.SetTotal == 0},
		{"rarity", p.Rarity == ""},
		{"releaseDate", p.ReleaseDate.IsZero()},
		{"illustrator", p.Illustrator == ""},
		{"image", p.ImageBase == ""},
	} {
		if f.empty {
			missing = append(missing, f.name)
		}
	}
	return missing
}

func preferName(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func preferCount(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}
