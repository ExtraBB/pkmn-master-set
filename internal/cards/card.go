// Package cards holds the product's domain model.
//
// The unit of this product is the *printing*, not the card: Base Set Charizard
// 1st Edition is a different thing to slot into a binder than Base Set Charizard
// Unlimited, so each gets its own row and its own placeholder. A Card is the
// physical card as the data source models it; a Printing is one row of output.
package cards

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Language is a card language. Only English and Japanese are in scope.
type Language string

const (
	LangEN Language = "en"
	LangJA Language = "ja"
)

// Valid reports whether l is a language this product supports.
func (l Language) Valid() bool { return l == LangEN || l == LangJA }

// Date is a calendar date with no time or zone. The zero value means "unknown",
// which is a real state: a few promo sets ship without a release date and the
// product shows that rather than inventing one.
type Date struct{ t time.Time }

const dateLayout = "2006-01-02"

// ParseDate accepts "YYYY-MM-DD". An empty string yields the zero Date without
// an error — unknown is a valid answer, not a parse failure.
func ParseDate(s string) (Date, error) {
	if s == "" {
		return Date{}, nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return Date{}, err
	}
	return Date{t: t}, nil
}

func (d Date) IsZero() bool { return d.t.IsZero() }

// String returns "YYYY-MM-DD", or "" when the date is unknown.
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.t.Format(dateLayout)
}

// Year returns the release year, or 0 when unknown.
func (d Date) Year() int {
	if d.IsZero() {
		return 0
	}
	return d.t.Year()
}

func (d Date) Before(o Date) bool { return d.t.Before(o.t) }

func (d Date) MarshalJSON() ([]byte, error) { return []byte(`"` + d.String() + `"`), nil }

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" {
		s = ""
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Set is a card set. ReleaseDate drives the chronological ordering that makes a
// filled binder read as the Pokémon's history, front to back.
type Set struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Series        string `json:"series"`
	ReleaseDate   Date   `json:"releaseDate"`
	CountOfficial int    `json:"countOfficial"`
}

// Card is one physical card, before variant expansion.
type Card struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Number      string    `json:"number"`
	DexIDs      []int     `json:"dexIds"`
	Rarity      string    `json:"rarity"`
	Illustrator string    `json:"illustrator"`
	ImageBase   string    `json:"imageBase"`
	SetID       string    `json:"setId"`
	SetName     string    `json:"setName"`
	SetTotal    int       `json:"setTotal"`
	Variants    []Variant `json:"variants"`
	// Pricing is the card's own pricing block, read from tcgdex's top-level
	// "pricing" field. It is the primary source for Printing.Pricing: it carries
	// every finish tcgdex has a listing for, and tcgdex populates it far more
	// reliably than any single variant's own copy — see Variant.Pricing.
	Pricing Pricing `json:"pricing"`
}

// CardmarketPricing is a card's Cardmarket average, in EUR. AvgHolo is the same
// average for the card's holo finish; Cardmarket does not price any finer than
// plain vs. holo.
type CardmarketPricing struct {
	ProductID int     `json:"productId"`
	Avg       float64 `json:"avg"`
	AvgHolo   float64 `json:"avgHolo"`
}

// TCGPlayerFinish is one priced finish of a card on TCGplayer, e.g. "normal" or
// "reverse-holofoil".
type TCGPlayerFinish struct {
	ProductID   int     `json:"productId"`
	MarketPrice float64 `json:"marketPrice"`
}

// TCGPlayerPricing is a card's TCGplayer prices, in USD, keyed by finish name.
// The finish vocabulary is not a fixed enum, so callers pick a finish rather than
// assuming one exists.
type TCGPlayerPricing struct {
	Finishes map[string]TCGPlayerFinish `json:"finishes"`
}

// Pricing is a market pricing block: either a card's own, or (occasionally) a
// specific variant's more specific copy of it. See Card.Pricing and
// Variant.Pricing for which one a Printing actually renders.
type Pricing struct {
	Cardmarket CardmarketPricing `json:"cardmarket"`
	TCGPlayer  TCGPlayerPricing  `json:"tcgplayer"`
}

// IsZero reports whether a pricing block carries no data at all.
func (p Pricing) IsZero() bool {
	return p.Cardmarket == (CardmarketPricing{}) && len(p.TCGPlayer.Finishes) == 0
}

// Printing is one row of output: a card in one specific printing. Everything the
// preview, the PDF, the CSV and the printable sheet need is denormalized onto it,
// so all four render the same fields from the same struct.
type Printing struct {
	// ID is stable across refetches: card ID plus the semantic variant key. The
	// source's own variantId is not stable and is deliberately unused.
	ID     string `json:"id"`
	CardID string `json:"cardId"`

	Name        string   `json:"name"`
	SetID       string   `json:"setId"`
	SetName     string   `json:"setName"`
	Series      string   `json:"series"`
	Number      string   `json:"number"`
	SetTotal    int      `json:"setTotal"`
	Rarity      string   `json:"rarity"`
	Variant     Variant  `json:"variant"`
	Language    Language `json:"language"`
	ReleaseDate Date     `json:"releaseDate"`
	Illustrator string   `json:"illustrator"`
	ImageBase   string   `json:"imageBase"`
	// Pricing is this printing's resolved market pricing: the variant's own
	// pricing when tcgdex has one, otherwise the card's. Zero when neither does.
	Pricing Pricing `json:"pricing"`

	// VariantLabel is the full human label ("1st Edition · Shadowless · Holo").
	// Empty when variants are collapsed, because the row then stands for the card
	// rather than for one printing.
	VariantLabel string `json:"variantLabel"`
	// VariantShort is the caption that fits on a 63x88mm placeholder.
	VariantShort string `json:"variantShort"`

	// Collapsed is how many printings this row stands for when variants are off.
	// It is 1 with variants on. Surfacing it keeps the variants toggle honest:
	// turning variants off folds rows away rather than deleting cards.
	Collapsed int `json:"collapsed"`

	// Missing names the fields the source had no value for, so downstream output
	// can print "unknown" instead of a blank cell. A blank cell reads as a bug;
	// an explicit "unknown" reads as a known gap.
	Missing []string `json:"missing,omitempty"`
}

// Number/total as collectors write it, e.g. "4/102".
func (p Printing) NumberOfTotal() string {
	if p.SetTotal == 0 {
		return p.Number
	}
	return p.Number + "/" + strconv.Itoa(p.SetTotal)
}

// ImageURL builds a tcgdex asset URL. The API returns a base path with no
// extension; quality is "low" or "high" and ext is "webp", "png" or "jpg".
// Returns "" when the source has no image for this card.
func (p Printing) ImageURL(quality, ext string) string {
	if p.ImageBase == "" {
		return ""
	}
	return p.ImageBase + "/" + quality + "." + ext
}

// CardmarketPrice is the Cardmarket average price for this printing, in EUR, and
// whether the source had one at all. Holo and reverse-holo finishes get the
// holo average; Cardmarket does not price any finer than that.
func (p Printing) CardmarketPrice() (float64, bool) {
	cm := p.Pricing.Cardmarket
	if (p.Variant.Type == "holo" || p.Variant.Type == "reverse") && cm.AvgHolo > 0 {
		return cm.AvgHolo, true
	}
	if cm.Avg > 0 {
		return cm.Avg, true
	}
	return 0, false
}

// CardmarketURL links to this printing's Cardmarket listing, or "" when the
// source had no product ID for it.
//
// Cardmarket's own redirect form is "/{Game}/Products?idProduct={id}" — no
// "/Singles" segment. That segment belongs to the browsable category listing,
// which is why appending idProduct to it just lands on the general Singles page
// instead of the product.
func (p Printing) CardmarketURL() string {
	if p.Pricing.Cardmarket.ProductID == 0 {
		return ""
	}
	q := url.Values{"idProduct": {strconv.Itoa(p.Pricing.Cardmarket.ProductID)}}
	return "https://www.cardmarket.com/en/Pokemon/Products?" + q.Encode()
}

// tcgplayerFinish picks the TCGplayer finish that best matches this printing's
// variant. TCGplayer's finish names are not this product's variant taxonomy, so
// the match is a best-effort guess by name, falling back to whatever finish the
// source has when nothing matches.
func (p Printing) tcgplayerFinish() (TCGPlayerFinish, bool) {
	finishes := p.Pricing.TCGPlayer.Finishes
	if len(finishes) == 0 {
		return TCGPlayerFinish{}, false
	}

	var prefix string
	switch {
	case slices.Contains(p.Variant.Stamps, "1st-edition"):
		prefix = "1st-edition"
	case p.Variant.Subtype == "unlimited":
		prefix = "unlimited"
	}
	var suffix string
	switch p.Variant.Type {
	case "holo":
		suffix = "holofoil"
	case "reverse":
		suffix = "reverse-holofoil"
	case "normal", "":
		suffix = "normal"
	}

	for _, key := range []string{joinNonEmpty(prefix, suffix), prefix, suffix, "normal"} {
		if key == "" {
			continue
		}
		if f, ok := finishes[key]; ok {
			return f, true
		}
	}

	// Nothing matched: any finish beats no price at all.
	keys := make([]string, 0, len(finishes))
	for k := range finishes {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return finishes[keys[0]], true
}

func joinNonEmpty(parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "-")
}

// TCGPlayerPrice is the TCGplayer market price for this printing's best-matched
// finish, in USD, and whether the source had one at all.
func (p Printing) TCGPlayerPrice() (float64, bool) {
	f, ok := p.tcgplayerFinish()
	if !ok || f.MarketPrice <= 0 {
		return 0, false
	}
	return f.MarketPrice, true
}

// TCGPlayerURL links to this printing's TCGplayer listing, or "" when the source
// had no product ID for it.
func (p Printing) TCGPlayerURL() string {
	f, ok := p.tcgplayerFinish()
	if !ok || f.ProductID == 0 {
		return ""
	}
	return "https://www.tcgplayer.com/product/" + strconv.Itoa(f.ProductID)
}

// bulbapediaGo is Bulbapedia's "jump to the article, else show search results"
// entry point. Going through search rather than straight at a title means a set
// this source spells differently lands on a search page, never on a red link.
const bulbapediaGo = "https://bulbapedia.bulbagarden.net/w/index.php"

// BulbapediaURL is where to go and read about this printing's card.
//
// Bulbapedia titles card articles "<Card> (<Set> <Number>)" — "Charizard (Base
// Set 4)" — but its expansion names do not always match this source's, which is
// why the link searches instead of deep-linking. Returns "" when the source lacks
// the name or the set the title is built from: Display renders those as the
// literal "unknown", and "unknown" is a thing to show a reader, never a thing to
// search for.
func (p Printing) BulbapediaURL() string {
	name := strings.TrimSpace(p.Name)
	set := strings.TrimSpace(p.SetName)
	if name == "" || set == "" {
		return ""
	}
	// The bare number, not the "4/102" the cell shows: the article title carries
	// the number alone.
	term := name + " (" + set
	if n := strings.TrimSpace(p.Number); n != "" {
		term += " " + n
	}
	term += ")"

	q := url.Values{"go": {"Go"}, "search": {term}, "title": {"Special:Search"}}
	return bulbapediaGo + "?" + q.Encode()
}

// DOMID is this printing's ID reduced to characters safe in an id attribute and a
// CSS selector. ID is the card ID plus Variant.Key(), which joins on "|" and "+".
// Rows reach the page in batches appended by htmx with no init hook, so per-row
// markup has to name its own controls from data that is already unique.
func (p Printing) DOMID() string {
	var b strings.Builder
	b.Grow(len(p.ID))
	for _, r := range p.ID {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			// One character for one character, so distinct keys stay distinct.
			b.WriteByte('-')
		}
	}
	return b.String()
}

// Lacks reports whether a named field was absent from the source, so templates
// can render "unknown" for it.
func (p Printing) Lacks(field string) bool { return slices.Contains(p.Missing, field) }

// Display renders one promised field for output.
//
// It exists so every output — preview, PDF, CSV, placeholder — writes the same
// text for the same data, including the literal "unknown" a missing field must
// show. A blank cell reads as a bug; a stated "unknown" reads as what it is, a gap
// in the source. See docs/variant-taxonomy.md.
//
// Variant is deliberately absent: with variants collapsed the row stands for the
// card rather than one printing, so its label is meaningfully empty rather than
// missing, and callers decide whether the column exists at all.
func (p Printing) Display(field string) string {
	const unknown = "unknown"
	switch field {
	case "name":
		return orUnknown(p.Name, unknown)
	case "set":
		return orUnknown(p.SetName, unknown)
	case "number":
		return orUnknown(p.NumberOfTotal(), unknown)
	case "rarity":
		return orUnknown(p.Rarity, unknown)
	case "illustrator":
		return orUnknown(p.Illustrator, unknown)
	case "language":
		return p.Language.Display()
	case "released":
		return orUnknown(p.ReleaseDate.String(), unknown)
	case "year":
		if p.ReleaseDate.IsZero() {
			return unknown
		}
		return strconv.Itoa(p.ReleaseDate.Year())
	case "cardmarketPrice":
		if v, ok := p.CardmarketPrice(); ok {
			return "€" + strconv.FormatFloat(v, 'f', 2, 64)
		}
		return unknown
	case "tcgplayerPrice":
		if v, ok := p.TCGPlayerPrice(); ok {
			return "$" + strconv.FormatFloat(v, 'f', 2, 64)
		}
		return unknown
	default:
		return ""
	}
}

func orUnknown(v, unknown string) string {
	if strings.TrimSpace(v) == "" {
		return unknown
	}
	return v
}

// Display names a language the way the UI says it.
func (l Language) Display() string {
	switch l {
	case LangEN:
		return "English"
	case LangJA:
		return "Japanese"
	default:
		return string(l)
	}
}

// Other returns the language a user would switch to. A list is one language or the
// other, never both — see docs/variant-taxonomy.md.
func (l Language) Other() Language {
	if l == LangJA {
		return LangEN
	}
	return LangJA
}
