package cards

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const taxonomyDoc = "../../docs/variant-taxonomy.md"

func readTaxonomy(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(taxonomyDoc))
	if err != nil {
		t.Fatalf("reading %s: %v", taxonomyDoc, err)
	}
	return string(b)
}

// The taxonomy is a promise to the user about what we counted. If the code learns
// to label a new kind of printing and the document doesn't mention it, the promise
// has quietly gone stale.
func TestTaxonomyDocumentsEveryLabelledCategory(t *testing.T) {
	doc := readTaxonomy(t)

	for _, label := range []string{
		"1st Edition", "Shadowless", "Unlimited",
		"Reverse Holo", "Holo",
		"Energy Symbol Error",
		"Jumbo",
	} {
		if !strings.Contains(doc, label) {
			t.Errorf("docs/variant-taxonomy.md never mentions %q, but the code labels printings with it", label)
		}
	}
}

// Excluding a category silently is the failure mode the issue calls out: a
// collector should never have to wonder whether we forgot something or left it
// out on purpose.
func TestTaxonomyNamesItsExclusions(t *testing.T) {
	doc := readTaxonomy(t)

	for _, topic := range []string{
		"Miscut",       // physical defects
		"Trainer",      // cards that merely depict a Pokémon
		"Prices",       // out of scope per the PRD
		"Grading",      // condition is not a printing
		"Japanese cov", // the known source gap
	} {
		if !strings.Contains(doc, topic) {
			t.Errorf("docs/variant-taxonomy.md does not address %q", topic)
		}
	}
}

// Each field the outputs render is promised by the taxonomy, and must be listed
// there so all four outputs agree on what a row contains.
func TestTaxonomyListsEveryPerCardField(t *testing.T) {
	doc := readTaxonomy(t)

	for _, field := range []string{
		"Card name", "Set", "Card number", "Rarity", "Variant", "Language",
		"Release date", "Illustrator",
	} {
		if !strings.Contains(doc, field) {
			t.Errorf("docs/variant-taxonomy.md does not list the %q field", field)
		}
	}
}

// The rules the code actually implements, spelled out in the document.
func TestTaxonomyStatesItsAttributionRules(t *testing.T) {
	doc := readTaxonomy(t)

	if !strings.Contains(doc, "both lists") {
		t.Error("the document does not state that dual-Pokémon cards appear in both lists")
	}
	if !strings.Contains(doc, "never both") {
		t.Error("the document does not state that a list is one language, never both")
	}
	if !strings.Contains(doc, "sort to the **end**") {
		t.Error("the document does not state where undated sets are ordered")
	}
}
