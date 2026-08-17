package cards

import (
	"encoding/json"
	"io"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Species is one Pokémon, identified by its National Dex number.
//
// The species list is not derived from card names: cards are titled "Charizard
// ex", "Dark Charizard", "M Charizard EX" and "Blaine's Charizard", none of which
// yield a canonical species name. It comes from a generated index instead.
type Species struct {
	DexID  int    `json:"dexId"`
	Name   string `json:"name"`
	NameJA string `json:"nameJa,omitempty"`

	// keys are precomputed folded forms of every name, so search does no work
	// per query beyond folding the query itself.
	keys []string
}

// Slug is the Pokémon's identity in a URL: /charizard, /mr-mime, /nidoran-f.
//
// It is readable rather than minimal — a shareable link is something a collector
// pastes to another collector, so "mr-mime" beats "mrmime". Lookups fold the slug
// back down (see ByName), so both forms resolve to the same Pokémon.
func (s Species) Slug() string { return slugify(s.Name) }

// slugify hyphenates a display name, reusing Fold for the accents and symbols so
// the slug and the search key can never disagree about a character.
//
//	Mr. Mime -> mr-mime      Farfetch'd -> farfetchd
//	Ho-Oh    -> ho-oh        Nidoran♀   -> nidoran-f
//	Type: Null -> type-null  Flabébé    -> flabebe
func slugify(name string) string {
	var parts []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			parts = append(parts, word.String())
			word.Reset()
		}
	}
	for _, r := range name {
		switch r {
		case ' ', '-':
			flush()
		case '♀', '♂':
			// A gender symbol is a word of its own, so Nidoran♀ reads as
			// "nidoran-f" rather than the run-on "nidoranf".
			flush()
			word.WriteString(Fold(string(r)))
			flush()
		case '.', ':', '_', '\'', '’':
			// Dropped, matching Fold: punctuation is not a word boundary a
			// collector would type.
		default:
			word.WriteString(Fold(string(r)))
		}
	}
	flush()
	return strings.Join(parts, "-")
}

// SpeciesIndex answers typeahead queries.
type SpeciesIndex struct {
	all    []Species
	byDex  map[int]Species
	byName map[string]Species
}

func NewSpeciesIndex(all []Species) *SpeciesIndex {
	idx := &SpeciesIndex{
		all:    make([]Species, len(all)),
		byDex:  make(map[int]Species, len(all)),
		byName: make(map[string]Species, len(all)),
	}
	copy(idx.all, all)
	for i := range idx.all {
		s := &idx.all[i]
		s.keys = s.keys[:0]
		for _, n := range []string{s.Name, s.NameJA} {
			if f := Fold(n); f != "" && !slices.Contains(s.keys, f) {
				s.keys = append(s.keys, f)
			}
		}
		idx.byDex[s.DexID] = *s
		for _, k := range s.keys {
			// First writer wins, so a later duplicate name cannot shadow an
			// earlier Pokémon's own URL.
			if _, taken := idx.byName[k]; !taken {
				idx.byName[k] = *s
			}
		}
	}
	slices.SortFunc(idx.all, func(a, b Species) int { return a.DexID - b.DexID })
	return idx
}

// LoadSpecies reads the generated species index.
func LoadSpecies(r io.Reader) (*SpeciesIndex, error) {
	var all []Species
	if err := json.NewDecoder(r).Decode(&all); err != nil {
		return nil, err
	}
	return NewSpeciesIndex(all), nil
}

func (idx *SpeciesIndex) ByDexID(dexID int) (Species, bool) {
	s, ok := idx.byDex[dexID]
	return s, ok
}

// ByName resolves a name or slug to one Pokémon, exactly.
//
// It folds the query, so every spelling of the same name lands on the same
// Pokémon: "charizard", "Charizard", "mr-mime", "mrmime" and "Mr. Mime" all
// resolve. This is what makes a shared URL survive being retyped by hand.
func (idx *SpeciesIndex) ByName(q string) (Species, bool) {
	s, ok := idx.byName[Fold(q)]
	return s, ok
}

func (idx *SpeciesIndex) Len() int { return len(idx.all) }

// Search returns Pokémon matching q, best match first.
//
// Prefix matches rank above substring matches so typing "char" surfaces Charizard,
// Charmander and Charmeleon before Wartortle-style incidental matches. A numeric
// query is treated as a dex number.
func (idx *SpeciesIndex) Search(q string, limit int) []Species {
	if limit <= 0 {
		return nil
	}

	if n, err := strconv.Atoi(strings.TrimSpace(q)); err == nil {
		if s, ok := idx.byDex[n]; ok {
			return []Species{s}
		}
		return nil
	}

	folded := Fold(q)
	if folded == "" {
		return nil
	}

	type hit struct {
		s    Species
		rank int
	}
	var hits []hit
	for _, s := range idx.all {
		best := -1
		for _, k := range s.keys {
			switch {
			case strings.HasPrefix(k, folded):
				best = 0
			case strings.Contains(k, folded):
				if best != 0 {
					best = 1
				}
			}
		}
		if best >= 0 {
			hits = append(hits, hit{s, best})
		}
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].rank != hits[j].rank {
			return hits[i].rank < hits[j].rank
		}
		return hits[i].s.DexID < hits[j].s.DexID
	})

	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]Species, len(hits))
	for i, h := range hits {
		out[i] = h.s
	}
	return out
}

// Fold normalises a name for forgiving search: lowercase, accents stripped,
// punctuation and spaces removed.
//
// Pokémon names use a small, closed set of awkward characters, so a hand-rolled
// mapper covers them all without pulling in golang.org/x/text:
//
//	Flabébé -> flabebe    Farfetch'd -> farfetchd
//	Mr. Mime -> mrmime    Nidoran♀   -> nidoranf
//	Type: Null -> typenull
//
// Japanese text passes through lowercased and otherwise untouched, so a JA query
// still matches a JA name exactly.
func Fold(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'á', 'à', 'â', 'ä', 'ã':
			b.WriteRune('a')
		case 'é', 'è', 'ê', 'ë':
			b.WriteRune('e')
		case 'í', 'ì', 'î', 'ï':
			b.WriteRune('i')
		case 'ó', 'ò', 'ô', 'ö', 'õ':
			b.WriteRune('o')
		case 'ú', 'ù', 'û', 'ü':
			b.WriteRune('u')
		case 'ñ':
			b.WriteRune('n')
		case 'ç':
			b.WriteRune('c')
		case '♀':
			b.WriteRune('f')
		case '♂':
			b.WriteRune('m')
		case ' ', '\'', '’', '.', ':', '-', '_':
			// dropped
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
