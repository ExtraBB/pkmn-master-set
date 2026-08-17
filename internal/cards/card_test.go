package cards

import (
	"strings"
	"testing"
)

// The set name in the preview is the way out of this product and into the card's
// history. It goes through Bulbapedia's search-and-go rather than at an article
// title, because this source's set names do not always match Bulbapedia's — and a
// search page is a useful answer where a red link is not.
func TestBulbapediaURL(t *testing.T) {
	const base = "https://bulbapedia.bulbagarden.net/w/index.php"

	tests := []struct {
		name string
		p    Printing
		want string
	}{
		{
			name: "a card links at its Bulbapedia article title",
			p:    Printing{Name: "Charizard", SetName: "Base Set", Number: "4", SetTotal: 102},
			want: base + "?go=Go&search=Charizard+%28Base+Set+4%29&title=Special%3ASearch",
		},
		{
			// Dark Charizard is not Charizard's article.
			name: "the card's own name is used, not the species'",
			p:    Printing{Name: "Dark Charizard", SetName: "Team Rocket", Number: "4"},
			want: base + "?go=Go&search=Dark+Charizard+%28Team+Rocket+4%29&title=Special%3ASearch",
		},
		{
			name: "no number: the set alone still reaches a search page",
			p:    Printing{Name: "Charizard", SetName: "Wizards Promo"},
			want: base + "?go=Go&search=Charizard+%28Wizards+Promo%29&title=Special%3ASearch",
		},
		{
			name: "no set name: nothing to link",
			p:    Printing{Name: "Charizard"},
			want: "",
		},
		{
			name: "no card name: nothing to link",
			p:    Printing{SetName: "Base Set", Number: "4"},
			want: "",
		},
		{
			name: "a whitespace-only field counts as missing",
			p:    Printing{Name: "Charizard", SetName: "   "},
			want: "",
		},
	}

	for _, tc := range tests {
		got := tc.p.BulbapediaURL()
		if got != tc.want {
			t.Errorf("%s:\nBulbapediaURL() = %q\nwant              %q", tc.name, got, tc.want)
		}
		// Display prints the literal "unknown" for a missing field. That is a
		// thing to show a reader, never a thing to search for.
		if strings.Contains(got, "unknown") {
			t.Errorf("%s: %q searches for the word \"unknown\"", tc.name, got)
		}
	}

	// The article title carries the bare number; "4/102" is how the cell reads,
	// not how Bulbapedia names the page.
	if got := tests[0].p.BulbapediaURL(); strings.Contains(got, "4%2F102") {
		t.Errorf("the link carries the number as 4/102: %q", got)
	}
}

// Every row names its own zoom control, and rows arrive in htmx-appended batches
// with no init hook to fix up ids afterwards — so the id has to be safe and unique
// straight out of the data.
func TestDOMID(t *testing.T) {
	// Printing.ID is the card ID plus Variant.Key(), joined with "#", "|" and "+".
	const id = "base1-4#holo||1st-edition|standard"
	if got, want := (Printing{ID: id}).DOMID(), "base1-4-holo--1st-edition-standard"; got != want {
		t.Errorf("DOMID() = %q, want %q", got, want)
	}

	for _, id := range []string{
		"base1-4#holo||1st-edition|standard",
		"base1-4#holo|shadowless||standard",
		"base1-4#holo|unlimited||standard",
		"base1-4",
	} {
		got := (Printing{ID: id}).DOMID()
		if strings.ContainsAny(got, " #|+.:/") {
			t.Errorf("DOMID() = %q, which is not safe in an id or a selector", got)
		}
	}

	// Two printings of the same card must not collide, or one row's thumbnail
	// opens another row's card.
	a := (Printing{ID: "base1-4#holo|shadowless||standard"}).DOMID()
	b := (Printing{ID: "base1-4#holo|unlimited||standard"}).DOMID()
	if a == b {
		t.Errorf("two variants share the DOM id %q", a)
	}
}
