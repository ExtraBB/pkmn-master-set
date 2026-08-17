package tcgdex

import "encoding/json"

// Wire DTOs for the tcgdex API. Only the fields this product needs are declared;
// everything else is dropped at decode time.

// Set is a tcgdex set. ReleaseDate is empty for a handful of promo sets.
type Set struct {
	ID   string
	Name string
	// SeriesID is the serie's own id ("base", "swsh", "tcgp", …), which is stable
	// across languages in a way the display name is not, so it is what callers
	// filter on.
	SeriesID      string
	Series        string
	ReleaseDate   string // "YYYY-MM-DD", or "" when unknown
	CountOfficial int
	CountTotal    int
}

// Variant is one entry of a card's variants_detailed array — the unit this
// product treats as a distinct printing. Pricing is that entry's own "pricing"
// field, but tcgdex populates it for only a minority of variants — most carry
// the zero value even when the card overall is priced. Callers should fall back
// to the card's own Pricing when a variant's is empty; see Card.Pricing.
type Variant struct {
	Type    string
	Subtype string
	Stamp   []string
	Size    string
	Pricing Pricing
}

// Card is a single physical card, before variant expansion. Pricing is read
// from the card's own top-level "pricing" field, which tcgdex populates far
// more reliably than any single variant's copy of it (roughly 90% of cards vs.
// 15% of variants, by observation) and which carries every finish tcgdex has a
// listing for (e.g. both "normal" and "reverse-holofoil" under one card), so it
// is the primary source; a variant's own Pricing, when present, is preferred
// only because it can occasionally be more specific.
type Card struct {
	ID          string
	Name        string
	Number      string // tcgdex localId
	DexIDs      []int
	Rarity      string
	Illustrator string
	ImageBase   string // "" when tcgdex returns null
	SetID       string
	SetName     string
	SetTotal    int
	Variants    []Variant
	Pricing     Pricing
}

// CardmarketPricing is the Cardmarket slice of a card's pricing block, in EUR.
// AvgHolo is the same average but for the card's holo finish; tcgdex does not
// split Cardmarket pricing any finer than plain vs. holo.
type CardmarketPricing struct {
	IDProduct int
	Avg       float64
	AvgHolo   float64
}

// TCGPlayerFinish is one priced finish inside a card's TCGplayer block, e.g. the
// "normal" or "reverse-holofoil" entry.
type TCGPlayerFinish struct {
	ProductID   int
	MarketPrice float64
}

// TCGPlayerPricing is the TCGplayer slice of a card's pricing block, in USD.
// Finishes is keyed by TCGplayer's own finish names, which are not a fixed enum
// (they vary with the card's print history), so they are decoded generically
// rather than declared field by field.
type TCGPlayerPricing struct {
	Finishes map[string]TCGPlayerFinish
}

// Pricing is the subset of tcgdex's pricing block this product surfaces.
type Pricing struct {
	Cardmarket CardmarketPricing
	TCGPlayer  TCGPlayerPricing
}

// IsZero reports whether a pricing block carries no data at all — the source
// had no listing to report.
func (p Pricing) IsZero() bool {
	return p.Cardmarket == (CardmarketPricing{}) && len(p.TCGPlayer.Finishes) == 0
}

// ---- REST wire shapes ----

type restSet struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ReleaseDate string `json:"releaseDate"`
	Serie       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"serie"`
	CardCount struct {
		Official int `json:"official"`
		Total    int `json:"total"`
	} `json:"cardCount"`
}

// briefRef matches the trimmed shape the REST list endpoints return for both
// cards and sets — only the id is useful, the rest needs a detail request.
type briefRef struct {
	ID string `json:"id"`
}

type restCard struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	LocalID     string        `json:"localId"`
	DexID       []int         `json:"dexId"`
	Rarity      string        `json:"rarity"`
	Illustrator string        `json:"illustrator"`
	Image       string        `json:"image"`
	Variants    []restVariant `json:"variants_detailed"`
	Pricing     restPricing   `json:"pricing"`
	Set         struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		CardCount struct {
			Official int `json:"official"`
			Total    int `json:"total"`
		} `json:"cardCount"`
	} `json:"set"`
}

// restVariant is the wire shape of one variants_detailed entry. Its pricing is
// decoded on its own terms (restPricing) and converted per entry, because each
// entry's price is that specific printing's price, not the card's.
type restVariant struct {
	Type    string      `json:"type"`
	Subtype string      `json:"subtype"`
	Stamp   []string    `json:"stamp"`
	Size    string      `json:"size"`
	Pricing restPricing `json:"pricing"`
}

func (rv restVariant) toVariant() Variant {
	return Variant{
		Type:    rv.Type,
		Subtype: rv.Subtype,
		Stamp:   rv.Stamp,
		Size:    rv.Size,
		Pricing: rv.Pricing.toPricing(),
	}
}

// restPricing mirrors the "pricing" object tcgdex attaches to a card. TCGplayer
// keys its finishes by name rather than by a fixed field set, so that side is
// decoded into a raw map and picked apart in toPricing.
type restPricing struct {
	Cardmarket struct {
		IDProduct int     `json:"idProduct"`
		Avg       float64 `json:"avg"`
		AvgHolo   float64 `json:"avg-holo"`
	} `json:"cardmarket"`
	TCGPlayer map[string]json.RawMessage `json:"tcgplayer"`
}

// tcgplayerMeta are the two TCGplayer keys that describe the block itself rather
// than naming a priced finish.
var tcgplayerMeta = map[string]bool{"unit": true, "updated": true}

func (p restPricing) toPricing() Pricing {
	out := Pricing{
		Cardmarket: CardmarketPricing{
			IDProduct: p.Cardmarket.IDProduct,
			Avg:       p.Cardmarket.Avg,
			AvgHolo:   p.Cardmarket.AvgHolo,
		},
	}
	for key, raw := range p.TCGPlayer {
		if tcgplayerMeta[key] {
			continue
		}
		var f struct {
			ProductID   int     `json:"productId"`
			MarketPrice float64 `json:"marketPrice"`
		}
		if err := json.Unmarshal(raw, &f); err != nil {
			continue
		}
		if out.TCGPlayer.Finishes == nil {
			out.TCGPlayer.Finishes = make(map[string]TCGPlayerFinish)
		}
		out.TCGPlayer.Finishes[key] = TCGPlayerFinish{ProductID: f.ProductID, MarketPrice: f.MarketPrice}
	}
	return out
}

func (c restCard) toCard() Card {
	variants := make([]Variant, len(c.Variants))
	for i, v := range c.Variants {
		variants[i] = v.toVariant()
	}
	return Card{
		ID:          c.ID,
		Name:        c.Name,
		Number:      c.LocalID,
		DexIDs:      c.DexID,
		Rarity:      normalizeRarity(c.Rarity),
		Illustrator: c.Illustrator,
		ImageBase:   c.Image,
		SetID:       c.Set.ID,
		SetName:     c.Set.Name,
		SetTotal:    c.Set.CardCount.Official,
		Variants:    variants,
		Pricing:     c.Pricing.toPricing(),
	}
}

func (s restSet) toSet() Set {
	return Set{
		ID:            s.ID,
		Name:          s.Name,
		SeriesID:      s.Serie.ID,
		Series:        s.Serie.Name,
		ReleaseDate:   s.ReleaseDate,
		CountOfficial: s.CardCount.Official,
		CountTotal:    s.CardCount.Total,
	}
}

// normalizeRarity maps tcgdex's sentinel "None" to an empty string, so downstream
// code has a single representation of "we don't know the rarity".
func normalizeRarity(r string) string {
	if r == "None" {
		return ""
	}
	return r
}
