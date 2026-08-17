//go:build live

package tcgdex

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"
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

	var resp struct {
		Cards []struct {
			ID       string `json:"id"`
			Category string `json:"category"`
			DexID    []int  `json:"dexId"`
		} `json:"cards"`
	}
	if err := c.graphQL(ctx, `{ cards { id category dexId } }`, nil, &resp); err != nil {
		t.Fatalf("fetching the corpus: %v", err)
	}

	var pokemon, untagged int
	bySet := map[string]int{}
	for _, card := range resp.Cards {
		if card.Category != "Pokemon" {
			continue
		}
		pokemon++
		if len(card.DexID) == 0 {
			untagged++
			bySet[setIDOf(card.ID)]++
		}
	}

	if pokemon == 0 {
		t.Fatal("no Pokémon cards returned")
	}

	sets := make([]string, 0, len(bySet))
	for id, n := range bySet {
		sets = append(sets, id)
		_ = n
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
