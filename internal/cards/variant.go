package cards

import (
	"slices"
	"strings"
)

// Variant is one distinct printing of a card.
//
// The taxonomy this product applies is deliberately inclusive: print runs,
// finishes, stamps and error prints all count as separate printings, because a
// collector filling a binder needs a physical slot for each. That makes the rule
// a one-liner — every distinct entry in the source's variant list is its own
// printing — which is defensible and easy to state. See docs/variant-taxonomy.md.
type Variant struct {
	// Type is the finish: "normal", "holo", "reverse", "lenticular", ...
	Type string `json:"type"`
	// Subtype is the print run or error qualifier: "shadowless", "unlimited",
	// "1999-2000-copyright", "energy-symbol-error", ...
	Subtype string `json:"subtype"`
	// Stamps are event/edition stamps, sorted: "1st-edition", "staff", "worlds-2010", ...
	Stamps []string `json:"stamps"`
	// Size is "standard" or "jumbo".
	Size string `json:"size"`
	// Pricing is this specific variant entry's own copy of the pricing block,
	// which tcgdex leaves zero for most variants even when the card overall is
	// priced — see Card.Pricing, which callers fall back to. Pricing plays no
	// part in Key() or dedup: two variants with the same identity but different
	// pricing data are still the same printing.
	Pricing Pricing `json:"-"`
}

// Key is the stable, semantic identity of a printing.
//
// The source assigns its own variantId, but those are unstable and it emits
// duplicate entries carrying different ones, so identity has to be derived from
// the meaning of the variant instead.
func (v Variant) Key() string {
	return strings.Join([]string{v.Type, v.Subtype, strings.Join(v.Stamps, "+"), v.Size}, "|")
}

// NormalizeVariant canonicalises a raw variant: lowercased tokens, sorted stamps,
// and "standard" filled in for a missing size.
func NormalizeVariant(typ, subtype string, stamps []string, size string) Variant {
	v := Variant{
		Type:    strings.ToLower(strings.TrimSpace(typ)),
		Subtype: strings.ToLower(strings.TrimSpace(subtype)),
		Size:    strings.ToLower(strings.TrimSpace(size)),
	}
	if v.Size == "" {
		v.Size = "standard"
	}
	for _, s := range stamps {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			v.Stamps = append(v.Stamps, s)
		}
	}
	slices.Sort(v.Stamps)
	v.Stamps = slices.Compact(v.Stamps)
	return v
}

// NormalizeVariants canonicalises, deduplicates and ranks a card's variants.
//
// Deduplication is by Key, never by the source's variantId: some cards come back
// with the same variant twice under different ids, and printing the same
// placeholder twice is exactly the kind of quiet wrongness this product exists to
// avoid.
//
// A card with no variants at all is treated as having a single plain printing, so
// it still occupies one binder slot.
func NormalizeVariants(vs []Variant) []Variant {
	seen := make(map[string]struct{}, len(vs))
	out := make([]Variant, 0, len(vs))
	for _, v := range vs {
		n := NormalizeVariant(v.Type, v.Subtype, v.Stamps, v.Size)
		n.Pricing = v.Pricing
		if _, dup := seen[n.Key()]; dup {
			continue
		}
		seen[n.Key()] = struct{}{}
		out = append(out, n)
	}
	if len(out) == 0 {
		out = append(out, NormalizeVariant("normal", "", nil, "standard"))
	}
	slices.SortFunc(out, compareVariant)
	return out
}

// ---- labelling ----

// printRunLabels are subtypes that describe which print run a card came from.
// These sort first in a label because they are what a collector says out loud:
// "Base Set Charizard 1st Edition", not "Base Set Charizard holo 1st Edition".
var printRunLabels = map[string]string{
	"1st-edition":          "1st Edition",
	"shadowless":           "Shadowless",
	"unlimited":            "Unlimited",
	"1999-2000-copyright":  "1999–2000 Copyright",
	"shadowless-red-cheek": "Shadowless Red Cheeks",
	"no-rarity":            "No Rarity Symbol",
	"blue-border":          "Blue Border",
}

// finishLabels describe how the card is printed. "normal" is deliberately absent:
// non-holo is the default and captioning it would waste space on a placeholder.
var finishLabels = map[string]string{
	"normal":     "",
	"holo":       "Holo",
	"reverse":    "Reverse Holo",
	"lenticular": "Lenticular",
}

// errorLabels are printing errors the source models as their own variant. Only
// errors that exist as a distinct, enumerable printing are here; physical defects
// like miscuts are excluded — see docs/variant-taxonomy.md.
var errorLabels = map[string]string{
	"energy-symbol-error": "Energy Symbol Error",
	"1st-edition-error":   "1st Edition Error",
}

// stampLabels name the stamps seen most often. Values are the stamp's name only —
// labelParts appends the word "Stamp".
//
// The stamp vocabulary is open and long-tailed (event, championship and promo
// stamps keep being added), so unknown stamps are humanized rather than dropped:
// a collector must never silently lose a printing because we had not seen its
// stamp before.
var stampLabels = map[string]string{
	"1st-edition":            "1st Edition",
	"staff":                  "Staff",
	"set-logo":               "Set Logo",
	"prerelease":             "Prerelease",
	"pokemon-league":         "League",
	"player-rewards-program": "Player Rewards",
	"pokemon-center":         "Pokémon Center",
	"winner":                 "Winner",
	"finalist":               "Finalist",
	"semi-finalist":          "Semi-Finalist",
	"quarter-finalist":       "Quarter-Finalist",
}

// sizeLabels: only a non-standard size is worth captioning.
var sizeLabels = map[string]string{
	"standard": "",
	"jumbo":    "Jumbo",
}

// labelDefault is used when a variant has nothing distinguishing to say.
const labelDefault = "Standard"

// Label is the full human label, e.g. "1st Edition · Shadowless · Holo".
func (v Variant) Label() string {
	parts := v.labelParts()
	if len(parts) == 0 {
		return labelDefault
	}
	return strings.Join(parts, " · ")
}

// Short is the one-phrase caption for a 63x88mm placeholder, where the full label
// would not fit. It answers "which printing is this?" with the most
// collector-salient part.
func (v Variant) Short() string {
	parts := v.labelParts()
	if len(parts) == 0 {
		return labelDefault
	}
	return parts[0]
}

// labelParts builds the label in collector order: print run, finish, stamps, size.
//
// A printing can carry two print-run facts at once — Base Set Charizard exists as
// Shadowless (subtype) *and* 1st Edition (stamp) on the same card — so both are
// emitted, 1st Edition first, giving "1st Edition · Shadowless · Holo".
func (v Variant) labelParts() []string {
	var parts []string

	if slices.Contains(v.Stamps, "1st-edition") {
		parts = append(parts, printRunLabels["1st-edition"])
	}
	switch {
	case printRunLabels[v.Subtype] != "":
		parts = append(parts, printRunLabels[v.Subtype])
	case errorLabels[v.Subtype] != "":
		parts = append(parts, errorLabels[v.Subtype])
	case v.Subtype != "":
		// Unknown subtype: humanize rather than drop.
		parts = append(parts, humanize(v.Subtype))
	}
	if s, ok := finishLabels[v.Type]; ok {
		if s != "" {
			parts = append(parts, s)
		}
	} else if v.Type != "" {
		parts = append(parts, humanize(v.Type))
	}
	for _, st := range v.Stamps {
		// A 1st Edition stamp is already reported as the print run.
		if st == "1st-edition" {
			continue
		}
		if s, ok := stampLabels[st]; ok {
			parts = append(parts, s+" Stamp")
		} else {
			parts = append(parts, humanize(st)+" Stamp")
		}
	}
	if s, ok := sizeLabels[v.Size]; ok {
		if s != "" {
			parts = append(parts, s)
		}
	} else if v.Size != "" {
		parts = append(parts, humanize(v.Size))
	}

	return parts
}

// humanize turns an unseen slug into something readable: "worlds-2010" becomes
// "Worlds 2010". Imperfect on purpose — visibly odd beats silently missing.
func humanize(slug string) string {
	words := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	for i, w := range words {
		if w == "" {
			continue
		}
		r := []rune(w)
		// Leave tokens that start with a digit alone ("2010", "1st").
		if r[0] < '0' || r[0] > '9' {
			r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		}
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// ---- ordering ----

// compareVariant orders a card's printings the way a collector would lay them out:
// earliest print run first, then plainest finish, then unstamped before stamped,
// then standard size before oversized.
func compareVariant(a, b Variant) int {
	if c := printRunRank(a) - printRunRank(b); c != 0 {
		return c
	}
	if c := finishRank(a.Type) - finishRank(b.Type); c != 0 {
		return c
	}
	if c := len(a.Stamps) - len(b.Stamps); c != 0 {
		return c
	}
	if c := strings.Compare(strings.Join(a.Stamps, "+"), strings.Join(b.Stamps, "+")); c != 0 {
		return c
	}
	if c := sizeRank(a.Size) - sizeRank(b.Size); c != 0 {
		return c
	}
	return strings.Compare(a.Key(), b.Key())
}

func printRunRank(v Variant) int {
	switch {
	case slices.Contains(v.Stamps, "1st-edition"), v.Subtype == "1st-edition":
		return 0
	case v.Subtype == "shadowless", v.Subtype == "shadowless-red-cheek", v.Subtype == "no-rarity":
		return 1
	case v.Subtype == "", v.Subtype == "unlimited":
		return 2
	default:
		return 3
	}
}

func finishRank(t string) int {
	switch t {
	case "normal":
		return 0
	case "holo":
		return 1
	case "reverse":
		return 2
	default:
		return 3
	}
}

func sizeRank(s string) int {
	if s == "jumbo" {
		return 1
	}
	return 0
}

// ---- collapsing ----

// Collapse picks the single variant that represents a card when the user turns
// variants off.
//
// It prefers the plainest printing — no special print run, no stamp — because
// that is the card a collector pictures when they say "I need the Base Set
// Charizard". A card that only ever existed stamped or 1st Edition has no plain
// printing, so the first ranked variant stands in.
func Collapse(vs []Variant) Variant {
	if len(vs) == 0 {
		return NormalizeVariant("normal", "", nil, "standard")
	}
	best := -1
	for i, v := range vs {
		if len(v.Stamps) > 0 || printRunRank(v) != 2 || v.Size != "standard" {
			continue
		}
		if best == -1 || finishRank(v.Type) < finishRank(vs[best].Type) {
			best = i
		}
	}
	if best == -1 {
		return vs[0]
	}
	return vs[best]
}
