//go:build live

package tcgdex

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"
)

// The corpus-wide queries this test needs. Both lean on REST filters rather than
// downloading every card and counting locally: `category=eq:Pokemon` narrows to
// cards that ought to carry a Dex number, and `dexId=null:` picks out the ones
// that do not.
const (
	pathAllPokemon      = "/en/cards?category=eq:Pokemon"
	pathUntaggedPokemon = "/en/cards?category=eq:Pokemon&dexId=null:"
)

// Cards are attributed to a Pokémon by their dexId, so a Pokémon card the source
// has not tagged with one is invisible to us — it cannot appear in any list, and
// a collector would never know it was missing.
//
// This test measures that blind spot. It is not a pass/fail on the source's
// tagging quality so much as an alarm: a small tail of freshly released cards is
// normal and self-corrects, but a large or growing gap means lists are quietly
// incomplete and the attribution rule needs rethinking.
//
//	go test -tags=live -run TestUntagged -v ./internal/tcgdex/
func TestUntaggedPokemonCardsStaySmall(t *testing.T) {
	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var all, missing []briefRef
	if err := c.get(ctx, pathAllPokemon, &all); err != nil {
		t.Fatalf("counting Pokémon cards: %v", err)
	}
	if err := c.get(ctx, pathUntaggedPokemon, &missing); err != nil {
		t.Fatalf("counting untagged Pokémon cards: %v", err)
	}

	pokemon, untagged := len(all), len(missing)
	if pokemon == 0 {
		// An empty list here is far more likely to be a filter the API stopped
		// recognising than a corpus with no Pokémon in it: an unknown query
		// parameter is treated as a filter matching nothing, so a renamed one fails
		// as a silent zero. Zero would also sail past the threshold below.
		t.Fatalf("no Pokémon cards returned for %s — has the filter syntax changed?", pathAllPokemon)
	}

	bySet := map[string]int{}
	for _, card := range missing {
		bySet[setIDOf(card.ID)]++
	}
	sets := make([]string, 0, len(bySet))
	for id := range bySet {
		sets = append(sets, id)
	}
	sort.Strings(sets)

	pct := 100 * float64(untagged) / float64(pokemon)
	t.Logf("%d of %d Pokémon cards (%.2f%%) have no Dex number and cannot be attributed", untagged, pokemon, pct)
	for _, id := range sets {
		t.Logf("  %s: %d cards", id, bySet[id])
	}

	// 1% is comfortably above the tail of a just-released set and well below the
	// point where lists would be meaningfully incomplete.
	if pct > 1.0 {
		t.Errorf("%.2f%% of Pokémon cards are unattributable, want under 1%%; affected sets: %s",
			pct, strings.Join(sets, ", "))
	}
}

func setIDOf(cardID string) string {
	if i := strings.LastIndex(cardID, "-"); i > 0 {
		return cardID[:i]
	}
	return cardID
}
