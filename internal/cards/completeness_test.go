//go:build live

package cards

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// completenessFixtures names the heavily-collected Pokémon whose lists are
// checked against a collector-sourced reference.
var completenessFixtures = []struct {
	dexID int
	name  string
	file  string
}{
	{6, "Charizard", "charizard.txt"},
	{25, "Pikachu", "pikachu.txt"},
	{133, "Eevee", "eevee.txt"},
	{150, "Mewtwo", "mewtwo.txt"},
	{197, "Umbreon", "umbreon.txt"},
}

// TestCompletenessAgainstCollectorReference is the check the whole product rests
// on: for the Pokémon people actually collect, does our list contain every card a
// collector's own reference lists?
//
// It is a subset check, not an equality check. Extra cards are fine — the
// reference lists are hand-maintained and lag new sets — but a card the reference
// has and we don't is a collector cutting out a set of placeholders with a hole
// in it, which is the failure this product exists to prevent.
//
// Fixtures are transcribed by hand from Bulbapedia's `<Pokémon>_(TCG)` pages; see
// testdata/completeness/README.md.
func TestCompletenessAgainstCollectorReference(t *testing.T) {
	cat := liveCatalog(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for _, fx := range completenessFixtures {
		t.Run(fx.name, func(t *testing.T) {
			want, err := readReference(filepath.Join("testdata", "completeness", fx.file))
			if err != nil {
				t.Fatalf("reading reference: %v", err)
			}
			if len(want) == 0 {
				t.Fatalf("reference %s is empty", fx.file)
			}

			got, err := cat.Query(ctx, Query{DexID: fx.dexID, Language: LangEN})
			if err != nil {
				t.Fatalf("Query: %v", err)
			}

			have := make(map[string]bool, len(got.Printings))
			for _, p := range got.Printings {
				have[cardKey(p.SetID, p.Number)] = true
			}

			var missing []string
			for _, ref := range want {
				if !have[ref] {
					missing = append(missing, ref)
				}
			}

			t.Logf("%s: %d cards in our list, %d in the reference", fx.name, len(have), len(want))
			if len(missing) > 0 {
				t.Errorf("%d card(s) in the collector reference are missing from our list:\n  %s",
					len(missing), strings.Join(missing, "\n  "))
			}
		})
	}
}

// readReference parses a fixture: one "setID number" per line, with # comments
// and blank lines ignored.
func readReference(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []string
	scan := bufio.NewScanner(f)
	for line := 1; scan.Scan(); line++ {
		text := strings.TrimSpace(scan.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) < 2 {
			return nil, fmt.Errorf("%s:%d: want \"setID number\", got %q", path, line, text)
		}
		out = append(out, cardKey(fields[0], fields[1]))
	}
	return out, scan.Err()
}

func cardKey(setID, number string) string {
	// Card numbers are written both padded and bare depending on the set, so
	// compare on the numeric value rather than the literal text.
	return strings.ToLower(setID) + " " + strings.TrimLeft(number, "0")
}
