package tcgdex

// Wire DTOs for the tcgdex API. Only the fields this product needs are declared;
// everything else (notably `pricing`, which the PRD explicitly excludes) is dropped
// at decode time.

// Set is a tcgdex set. ReleaseDate is empty for a handful of promo sets.
type Set struct {
	ID            string
	Name          string
	Series        string
	ReleaseDate   string // "YYYY-MM-DD", or "" when unknown
	CountOfficial int
	CountTotal    int
}

// Variant is one entry of a card's variants_detailed array — the unit this
// product treats as a distinct printing.
type Variant struct {
	Type    string   `json:"type"`
	Subtype string   `json:"subtype"`
	Stamp   []string `json:"stamp"`
	Size    string   `json:"size"`
}

// Card is a single physical card, before variant expansion.
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
}

// ---- REST wire shapes ----

type restSet struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ReleaseDate string `json:"releaseDate"`
	Serie       struct {
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
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	LocalID     string    `json:"localId"`
	DexID       []int     `json:"dexId"`
	Rarity      string    `json:"rarity"`
	Illustrator string    `json:"illustrator"`
	Image       string    `json:"image"`
	Variants    []Variant `json:"variants_detailed"`
	Set         struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		CardCount struct {
			Official int `json:"official"`
			Total    int `json:"total"`
		} `json:"cardCount"`
	} `json:"set"`
}

func (c restCard) toCard() Card {
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
		Variants:    c.Variants,
	}
}

func (s restSet) toSet() Set {
	return Set{
		ID:            s.ID,
		Name:          s.Name,
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
