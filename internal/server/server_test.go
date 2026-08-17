package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ExtraBB/pkmn-master-set/internal/cards"
)

// fakeSource stands in for tcgdex so nothing here touches the network. The fixture
// is deliberately awkward: a card with no rarity, a set with no release date and a
// card with no art, because those are the cases the product promises to show
// honestly rather than as blanks.
type fakeSource struct {
	cards map[cards.Language][]cards.Card
	sets  map[string]cards.Set
	err   error
}

func (f *fakeSource) Cards(_ context.Context, lang cards.Language, dexID int) ([]cards.Card, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []cards.Card
	for _, c := range f.cards[lang] {
		for _, id := range c.DexIDs {
			if id == dexID {
				out = append(out, c)
				break
			}
		}
	}
	return out, nil
}

func (f *fakeSource) Set(_ context.Context, _ cards.Language, setID string) (cards.Set, error) {
	if f.err != nil {
		return cards.Set{}, f.err
	}
	s, ok := f.sets[setID]
	if !ok {
		return cards.Set{}, fmt.Errorf("no such set %q", setID)
	}
	return s, nil
}

func date(t *testing.T, s string) cards.Date {
	t.Helper()
	d, err := cards.ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", s, err)
	}
	return d
}

func variant(typ, subtype string, stamps ...string) cards.Variant {
	return cards.NormalizeVariant(typ, subtype, stamps, "standard")
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	return newTestServerWith(t, testSource(t))
}

func newTestServerWith(t *testing.T, src cards.Source) http.Handler {
	t.Helper()
	species := cards.NewSpeciesIndex([]cards.Species{
		{DexID: 4, Name: "Charmander"},
		{DexID: 5, Name: "Charmeleon"},
		{DexID: 6, Name: "Charizard"},
		{DexID: 25, Name: "Pikachu"},
		{DexID: 94, Name: "Gengar"},
		{DexID: 122, Name: "Mr. Mime"},
		{DexID: 133, Name: "Eevee"},
	})
	return New(cards.NewCatalog(src, species)).Routes()
}

// testSource is one Charizard's worth of fixture: 4 cards expanding to 7 printings.
func testSource(t *testing.T) *fakeSource {
	t.Helper()
	return &fakeSource{
		sets: map[string]cards.Set{
			"base1": {ID: "base1", Name: "Base Set", ReleaseDate: date(t, "1999-01-09"), CountOfficial: 102},
			"base2": {ID: "base2", Name: "Base Set 2", ReleaseDate: date(t, "2000-02-24"), CountOfficial: 130},
			"rocket": {ID: "rocket", Name: "Team Rocket", ReleaseDate: date(t, "2000-04-24"),
				CountOfficial: 82},
			// No release date: this set must sort to the end, not the start.
			"promo": {ID: "promo", Name: "Wizards Promo"},
		},
		cards: map[cards.Language][]cards.Card{
			cards.LangEN: {
				{
					ID: "base1-4", Name: "Charizard", Number: "4", DexIDs: []int{6},
					Rarity: "Holo Rare", Illustrator: "Mitsuhiro Arita", ImageBase: "https://img/base1/4",
					SetID: "base1", SetName: "Base Set", SetTotal: 102,
					Variants: []cards.Variant{
						variant("holo", "unlimited"),
						variant("holo", "shadowless"),
						variant("holo", "", "1st-edition"),
					},
				},
				{
					ID: "base2-4", Name: "Charizard", Number: "4", DexIDs: []int{6},
					Rarity: "Holo Rare", Illustrator: "Mitsuhiro Arita", ImageBase: "https://img/base2/4",
					SetID: "base2", SetName: "Base Set 2", SetTotal: 130,
					Variants: []cards.Variant{variant("holo", "unlimited")},
				},
				{
					ID: "rocket-4", Name: "Dark Charizard", Number: "4", DexIDs: []int{6},
					Rarity: "Holo Rare", Illustrator: "Ken Sugimori", ImageBase: "https://img/rocket/4",
					SetID: "rocket", SetName: "Team Rocket", SetTotal: 82,
					Variants: []cards.Variant{variant("holo", ""), variant("normal", "")},
				},
				{
					// No rarity, no illustrator, no art, no set date: every gap at once.
					ID: "promo-1", Name: "Charizard", Number: "1", DexIDs: []int{6},
					SetID: "promo", SetName: "Wizards Promo",
					Variants: []cards.Variant{variant("holo", "")},
				},
				{
					ID: "base1-46", Name: "Charmander", Number: "46", DexIDs: []int{4},
					Rarity: "Common", SetID: "base1", SetName: "Base Set", SetTotal: 102,
					Variants: []cards.Variant{variant("normal", "unlimited")},
				},
			},
			// Deliberately empty: Charizard has no Japanese cards in this fixture, which
			// is the "known Pokémon, no cards" state the UI must state honestly.
			cards.LangJA: {},
		},
	}
}

func get(t *testing.T, h http.Handler, path string, headers ...string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPages(t *testing.T) {
	h := newTestServer(t)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantAll    []string
		wantNone   []string
	}{
		{
			name:       "landing offers one search box and nothing to sign up for",
			path:       "/",
			wantStatus: http.StatusOK,
			wantAll: []string{
				`action="/go"`, `placeholder="Search a Pokémon`,
				`href="/charizard"`, "63 × 88 mm",
			},
			wantNone: []string{"sign up", "create an account", "log in", "price"},
		},
		{
			name:       "static assets are served",
			path:       "/static/css/app.css",
			wantStatus: http.StatusOK,
			wantAll:    []string{"--accent"},
		},
		{
			name:       "typeahead forgives a partial name",
			path:       "/search?q=char",
			wantStatus: http.StatusOK,
			wantAll: []string{
				"Charizard", "Charmander", "Charmeleon",
				`href="/charizard"`, "#006", `/static/sprites/6.png`,
			},
		},
		{
			name:       "typeahead is case and punctuation insensitive",
			path:       "/search?q=MR+MIME",
			wantStatus: http.StatusOK,
			wantAll:    []string{"Mr. Mime", `href="/mr-mime"`},
		},
		{
			name:       "typeahead says so when nothing matches",
			path:       "/search?q=zzz",
			wantStatus: http.StatusOK,
			wantAll:    []string{"No Pokémon matches"},
		},
		{
			name:       "an unknown Pokémon is a named miss, not an empty page",
			path:       "/notapokemon",
			wantStatus: http.StatusNotFound,
			wantAll:    []string{"notapokemon", `action="/go"`},
		},
		{
			name:       "a download format we do not serve is a 404",
			path:       "/download/charizard/xlsx",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, h, tt.path)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			body := rec.Body.String()
			for _, want := range tt.wantAll {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
			for _, unwanted := range tt.wantNone {
				if strings.Contains(strings.ToLower(body), unwanted) {
					t.Errorf("body contains %q, which has no place in this flow", unwanted)
				}
			}
		})
	}
}

// The preview is the confirmation step: the counts, the order and the fields all
// have to be the ones the download will use.
func TestPreviewWithVariants(t *testing.T) {
	rec := get(t, newTestServer(t), "/charizard")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	// 4 cards, 3+1+2+1 printings.
	for _, want := range []string{
		"7 cards", "≈ 1 page of placeholders",
		"7 cards · 1 page", "7 rows", // the download cards carry their own size
		"Base Set", "4/102", "Holo Rare", "1st Edition", "Shadowless", "Unlimited",
		"Mitsuhiro Arita", "1999",
		"Dark Charizard", // a card whose name is not the species name reads as itself
		"Print at 100%",  // the scale defence
		"50 mm",          // and something to measure
		"/download/charizard/sheets",
		"All 7 printings shown",
		"search=Charizard&#43;%28Base&#43;Set&#43;4%29", // the set name reads up on Bulbapedia
		`popovertarget="zoom-base1-4`,                   // and the thumbnail opens the card
	} {
		if !strings.Contains(body, want) {
			t.Errorf("preview does not contain %q", want)
		}
	}

	// Chronological by set release, undated sets last.
	wantOrder := []string{"Base Set", "Base Set 2", "Team Rocket", "Wizards Promo"}
	if !inOrder(body, wantOrder) {
		t.Errorf("sets are not in chronological order; want %v", wantOrder)
	}
	// Within Base Set, earliest print run first.
	if !inOrder(body, []string{"1st Edition", "Shadowless", "Unlimited"}) {
		t.Error("Base Set printings are not in print-run order")
	}

	// A gap in the source reads as "unknown", never as a blank cell.
	if strings.Count(body, "unknown") < 2 {
		t.Error("the promo card's missing rarity and illustrator should render as unknown")
	}

	// The preview is not a workspace.
	for _, unwanted := range []string{"checkbox", "Sort by", "Filter"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("preview contains %q, which turns it into a workspace", unwanted)
		}
	}
}

// Toggling variants has to move the list and the paper cost visibly, in one place,
// because the alternative is discovering it after printing.
func TestPreviewVariantsOff(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/charizard?variants=0").Body.String()

	for _, want := range []string{
		"4 cards",     // one row per card
		"−3 cards",    // the delta pill states what folding away cost
		"3 printings", // and each row says what it stands for
		// Turning variants back on is the canonical URL, with nothing to strip.
		`hx-get="/charizard"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("collapsed preview does not contain %q", want)
		}
	}
	// With variants collapsed a row stands for the card, so no printing label is
	// claimed for it.
	if strings.Contains(body, "Shadowless") {
		t.Error("a collapsed row must not be captioned with one printing's name")
	}
	if strings.Contains(body, ">Variant<") {
		t.Error("the variant column should give way to the printing count")
	}
}

// The set name is the way out of the preview and into the card's history. It goes
// through Bulbapedia's search-and-go rather than at an article title, because this
// source's set names do not always match Bulbapedia's and a search page is a useful
// answer where a red link is not.
func TestPreviewSetLinksToBulbapedia(t *testing.T) {
	body := get(t, newTestServer(t), "/charizard").Body.String()

	// html/template writes & as &amp; and + as &#43; inside an attribute.
	const base = "https://bulbapedia.bulbagarden.net/w/index.php?go=Go&amp;search="
	for _, want := range []string{
		`<a class="set-link" href="` + base + `Charizard&#43;%28Base&#43;Set&#43;4%29&amp;title=Special%3ASearch"`,
		base + `Dark&#43;Charizard&#43;%28Team&#43;Rocket&#43;4%29`, // the card's name, not the species'
		base + `Charizard&#43;%28Wizards&#43;Promo&#43;1%29`,        // no art and no rarity, still a real link
		`target="_blank"`, `rel="noopener noreferrer"`, // the preview is not lost to a wiki page
		"links to the card on Bulbapedia", // said once, in the column header
	} {
		if !strings.Contains(body, want) {
			t.Errorf("preview does not contain %q", want)
		}
	}

	// The article title carries the bare number, not the 4/102 the cell shows.
	if strings.Contains(body, "4%2F102") {
		t.Error("the link searches for the number as 4/102")
	}
	// Display renders a missing field as "unknown". That is a thing to show a
	// reader, never a thing to search for.
	if strings.Contains(body, "search=unknown") || strings.Contains(body, "%28unknown") {
		t.Error(`a link searches Bulbapedia for the word "unknown"`)
	}
}

// A 44px thumbnail cannot answer "is that the printing I think it is", so clicking
// it opens the card. It is HTML and CSS only: this site has no application
// JavaScript, and row batches are appended with no init hook to run afterwards.
func TestPreviewCardZoom(t *testing.T) {
	body := get(t, newTestServer(t), "/charizard").Body.String()

	for _, want := range []string{
		`popovertarget="zoom-base1-4-holo--1st-edition-standard"`,
		`id="zoom-base1-4-holo--1st-edition-standard" popover`,
		`aria-label="Enlarge Charizard, Base Set 4/102"`,
		// The enlarged art is a background image, so a closed lightbox never
		// fetches it.
		`style="background-image:url(https://img/base1/4/high.webp)"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("preview does not contain %q", want)
		}
	}

	// Six of the seven printings have art. The promo card has none, so it keeps its
	// marked-blank slot and gets no control that could not do anything.
	if got := strings.Count(body, "popovertarget="); got != 6 {
		t.Errorf("zoom controls = %d, want 6 (one per row with art)", got)
	}
	if !strings.Contains(body, "thumb-blank") {
		t.Error("the promo card has no art and must still get a marked-blank slot")
	}

	if strings.Contains(body, `src="https://img/base1/4/high`) {
		t.Error("a hidden <img> per row would queue a full-size fetch for every row")
	}
	if strings.Contains(body, "ZgotmplZ") {
		t.Error("an image URL was rejected by the template escaper")
	}
	// The panel itself carries no script and no inline handler: the only JavaScript
	// on this site is the vendored htmx in the layout.
	panel := get(t, newTestServer(t), "/charizard", "HX-Request", "true").Body.String()
	if strings.Contains(panel, "<script") || strings.Contains(panel, "onclick") {
		t.Error("the zoom must stay HTML and CSS only")
	}
}

// htmx swaps the panel so the counts and the list update together; the URL still
// has to be the shareable one.
func TestPreviewFragment(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/charizard?variants=0", "HX-Request", "true")
	body := rec.Body.String()

	if strings.Contains(body, "<html") {
		t.Error("an htmx request should get the panel, not the whole page")
	}
	for _, want := range []string{`id="preview"`, "4 cards", "−3 cards"} {
		if !strings.Contains(body, want) {
			t.Errorf("panel does not contain %q", want)
		}
	}
}

// The full list must be reachable — a preview that hides most of the list cannot
// answer the completeness question.
func TestRowsPagination(t *testing.T) {
	h := newTestServer(t)

	rec := get(t, h, "/rows/charizard?offset=6")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Wizards Promo") {
		t.Error("the last row should be in the last batch")
	}
	if strings.Contains(body, "Base Set 2") {
		t.Error("rows before the offset should not be repeated")
	}
	if !strings.Contains(body, `hx-swap-oob="true"`) {
		t.Error("the batch should refresh the footer count out of band")
	}
	if !strings.Contains(body, "All 7 printings shown") {
		t.Error("the refreshed footer should report that the list is complete")
	}
	// The promo card is the whole of this batch and has no art, so the batch has a
	// blank slot and no zoom control.
	if !strings.Contains(body, "thumb-blank") || strings.Contains(body, "popovertarget=") {
		t.Error("a row with no art should carry a blank slot and no zoom control")
	}
	if !strings.Contains(body, "set-link") {
		t.Error("appended rows should link to Bulbapedia like every other row")
	}

	// An appended batch gets no init hook, so it has to bring its own controls.
	batch := get(t, h, "/rows/charizard?offset=1").Body.String()
	for _, want := range []string{`popovertarget="zoom-`, "background-image:url(", "set-link"} {
		if !strings.Contains(batch, want) {
			t.Errorf("an appended batch does not contain %q", want)
		}
	}

	// An offset past the end is an empty batch, not a crash.
	if rec := get(t, h, "/rows/charizard?offset=999"); rec.Code != http.StatusOK {
		t.Errorf("offset past the end: status = %d, want 200", rec.Code)
	}
}

// A Pokémon with no cards in a language is an empty result, not a failure, and the
// page has to say which it is.
func TestPreviewEmptyLanguage(t *testing.T) {
	body := get(t, newTestServer(t), "/charizard?lang=ja").Body.String()

	for _, want := range []string{"No Japanese cards", "thinner", `href="/charizard"`} {
		if !strings.Contains(body, want) {
			t.Errorf("empty list does not contain %q", want)
		}
	}
	if strings.Contains(body, "Print at 100%") {
		t.Error("there is nothing to print, so the print instructions should be absent")
	}
}

// An unreachable source must never render as "this Pokémon has no cards".
func TestPreviewSourceUnavailable(t *testing.T) {
	src := testSource(t)
	src.err = errors.New("tcgdex is down")
	rec := get(t, newTestServerWith(t, src), "/charizard")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"can’t reach the card database", "Try Charizard again"} {
		if !strings.Contains(body, want) {
			t.Errorf("unavailable page does not contain %q", want)
		}
	}
	if strings.Contains(body, "0 cards") {
		t.Error("a failure must not be reported as an empty list")
	}
}

// Pressing Enter in the search box has to work without picking a suggestion, and
// without JavaScript.
func TestGoRedirects(t *testing.T) {
	h := newTestServer(t)

	tests := []struct {
		query      string
		wantStatus int
		wantTo     string
	}{
		{"Charizard", http.StatusSeeOther, "/charizard"},
		{"charizard", http.StatusSeeOther, "/charizard"},
		{"mr mime", http.StatusSeeOther, "/mr-mime"},
		{"char", http.StatusSeeOther, "/charmander"}, // best match: lowest dex number
		{"6", http.StatusSeeOther, "/charizard"},     // a dex number is a search too
		{"zzzz", http.StatusNotFound, ""},
	}
	// A Dex number typed straight into the path redirects to the one canonical URL.
	if rec := get(t, h, "/6?variants=0"); rec.Code != http.StatusSeeOther ||
		rec.Header().Get("Location") != "/charizard?variants=0" {
		t.Errorf("/6?variants=0 = %d %q, want 303 /charizard?variants=0",
			rec.Code, rec.Header().Get("Location"))
	}
	for _, tt := range tests {
		rec := get(t, h, "/go?q="+strings.ReplaceAll(tt.query, " ", "+"))
		if rec.Code != tt.wantStatus {
			t.Errorf("q=%q: status = %d, want %d", tt.query, rec.Code, tt.wantStatus)
			continue
		}
		if got := rec.Header().Get("Location"); got != tt.wantTo {
			t.Errorf("q=%q: Location = %q, want %q", tt.query, got, tt.wantTo)
		}
	}
}

// The two overview downloads arrive with issue 004. Their routes exist so the
// preview's buttons are real links rather than decoration.
func TestOverviewDownloadsNotYetImplemented(t *testing.T) {
	h := newTestServer(t)
	for _, format := range []string{"pdf", "csv"} {
		rec := get(t, h, "/download/charizard/"+format)
		if rec.Code != http.StatusNotImplemented {
			t.Errorf("%s: status = %d, want 501", format, rec.Code)
		}
	}
}

// ---- printable sheets ----

// The core output. Everything asserted here is something a collector finds out
// about only after cutting, if it is wrong.
func TestSheets(t *testing.T) {
	rec := get(t, newTestServer(t), "/download/charizard/sheets")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		// The scale defence, on the page and in the instructions.
		"50 mm", "Print at 100%", "fit to page",
		// The caption: set, number, printing.
		"Base Set", "4/102", "1st Edition", "Shadowless", "Unlimited",
		"Dark Charizard",                      // a card whose name is not the species' reads as itself
		`src="https://img/base1/4/high.webp"`, // image-led, at print quality
		`href="/charizard"`,                   // and a way back to the list
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sheet does not contain %q", want)
		}
	}

	// An image that has not loaded because it is off screen prints blank.
	if strings.Contains(body, `loading="lazy"`) {
		t.Error("placeholder art must not be lazily loaded")
	}
	// The sheet is a physical artifact: no site chrome, no prices, nothing to click
	// off the page.
	for _, unwanted := range []string{"htmx", "cardmarket", "tcgplayer", "€", "$"} {
		if strings.Contains(strings.ToLower(body), unwanted) {
			t.Errorf("sheet contains %q, which does not belong on paper", unwanted)
		}
	}

	// Same order as the preview: chronological by set release, undated last.
	if !inOrder(body, []string{"Base Set", "Base Set 2", "Team Rocket", "Wizards Promo"}) {
		t.Error("sheet printings are not in the preview's chronological order")
	}
	if !inOrder(body, []string{"1st Edition", "Shadowless", "Unlimited"}) {
		t.Error("Base Set printings are not in print-run order")
	}

	// One placeholder per printing, and no more paper than the preview promised.
	if got := strings.Count(body, `class="placeholder`); got != 7 {
		t.Errorf("placeholders = %d, want 7 (one per printing)", got)
	}
	if got := strings.Count(body, `class="sheet"`); got != 1 {
		t.Errorf("sheets = %d, want 1 — the preview promised 1 page", got)
	}

	// The promo card has no art in the source. With no picture to recognise, the
	// placeholder has to say everything a captioned one says — set, number and
	// printing, not just the name — because those words are all that identifies
	// the gap.
	blank, ok := blockAt(body, "art-blank")
	if !ok {
		t.Fatal("a card with no art must still get a placeholder")
	}
	for _, want := range []string{"Charizard", "Wizards Promo", "1", "Holo"} {
		if !strings.Contains(blank, want) {
			t.Errorf("the art-less placeholder does not contain %q", want)
		}
	}
	// The set is what you flip to, so it leads; the card names itself under it,
	// and the number and printing settle it in the smallest type.
	if !inOrder(blank, []string{"blank-set", "blank-name", "blank-meta"}) {
		t.Error("the art-less placeholder does not lead with the set")
	}
	if strings.Contains(body, `src=""`) || strings.Contains(body, "ZgotmplZ") {
		t.Error("a card with no art rendered an empty or rejected image URL")
	}
}

// The geometry is the acceptance criterion, and it lives in the stylesheet: a
// placeholder that measures 61 mm does not sit in a binder pocket.
func TestSheetGeometry(t *testing.T) {
	rec := get(t, newTestServer(t), "/static/css/print.css")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	css := rec.Body.String()

	for _, want := range []string{
		"width: 63mm", "height: 88mm", // true trading-card size
		"width: 189mm", // a 3 x 3 grid of them
		"height: 50mm", // the strip the user measures before cutting
		// A4, named so the print dialog opens on the right paper, with margins
		// small enough that the grid never has to be shrunk to fit.
		"@page { size: A4; margin: 4mm; }",
		"print-color-adjust", // or the cut lines and the greying never reach paper
	} {
		if !strings.Contains(css, want) {
			t.Errorf("print.css does not contain %q", want)
		}
	}
}

// The page count on the sheet has to be the one the preview promised, because
// that number is what the user decided to spend paper on.
func TestSheetsPageCountMatchesThePromise(t *testing.T) {
	src := testSource(t)
	// Eleven printings: two pages of nine, so the chunking is actually exercised.
	for i := range 10 {
		src.cards[cards.LangEN] = append(src.cards[cards.LangEN], cards.Card{
			ID: fmt.Sprintf("base2-%d", 20+i), Name: "Charizard", Number: fmt.Sprintf("%d", 20+i),
			DexIDs: []int{6}, Rarity: "Rare", ImageBase: fmt.Sprintf("https://img/base2/%d", 20+i),
			SetID: "base2", SetName: "Base Set 2", SetTotal: 130,
			Variants: []cards.Variant{variant("normal", "unlimited")},
		})
	}
	h := newTestServerWith(t, src)

	body := get(t, h, "/download/charizard/sheets").Body.String()
	if got := strings.Count(body, `class="placeholder`); got != 17 {
		t.Errorf("placeholders = %d, want 17", got)
	}
	if got := strings.Count(body, `class="sheet"`); got != 2 {
		t.Errorf("sheets = %d, want 2 (17 placeholders, 9 to a page)", got)
	}
	if got := strings.Count(get(t, h, "/charizard").Body.String(), "2 pages"); got == 0 {
		t.Error("the preview should have promised 2 pages for the same list")
	}
}

// All three treatments are reachable, and a treatment nobody asked for falls back
// to the default rather than failing: a mistyped parameter in a shared link should
// still print.
func TestSheetsArtTreatments(t *testing.T) {
	h := newTestServer(t)

	tests := []struct {
		path     string
		wantAll  []string
		wantNone []string
	}{
		{
			path:    "/download/charizard/sheets",
			wantAll: []string{`class="art-grey"`, `src="https://img/base1/4/high.webp"`},
		},
		{
			path:    "/download/charizard/sheets?art=colour",
			wantAll: []string{`class="art-colour"`, `src="https://img/base1/4/high.webp"`},
		},
		{
			// No art at all: the name carries the identification, and the sheet
			// costs a fraction of the ink.
			path:     "/download/charizard/sheets?art=outline",
			wantAll:  []string{`class="art-outline"`, "art-blank", "Dark Charizard"},
			wantNone: []string{"<img"},
		},
		{
			path:    "/download/charizard/sheets?art=nonsense",
			wantAll: []string{`class="art-grey"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := get(t, h, tt.path)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			body := rec.Body.String()
			for _, want := range tt.wantAll {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
			for _, unwanted := range tt.wantNone {
				if strings.Contains(body, unwanted) {
					t.Errorf("body contains %q", unwanted)
				}
			}
		})
	}
}

// The sheet carries the options it was asked for, so the paper matches the list
// the user was looking at.
func TestSheetsCarryTheOptions(t *testing.T) {
	h := newTestServer(t)

	body := get(t, h, "/download/charizard/sheets?variants=0").Body.String()
	if got := strings.Count(body, `class="placeholder`); got != 4 {
		t.Errorf("placeholders = %d, want 4 (one per card, variants off)", got)
	}
	// With variants collapsed the row stands for the card, so no printing is
	// claimed for it.
	if strings.Contains(body, "Shadowless") {
		t.Error("a collapsed placeholder must not be captioned with one printing's name")
	}
	// The options survive the round trip through the art picker and the back link.
	for _, want := range []string{`name="variants" value="0"`, `href="/charizard?variants=0"`} {
		if !strings.Contains(body, want) {
			t.Errorf("sheet does not contain %q", want)
		}
	}

	// A Pokémon with no cards in a language prints nothing, and says so rather than
	// spooling blank paper.
	empty := get(t, h, "/download/charizard/sheets?lang=ja")
	if empty.Code != http.StatusOK {
		t.Fatalf("empty list: status = %d, want 200", empty.Code)
	}
	if !strings.Contains(empty.Body.String(), "Nothing to print") {
		t.Error("an empty list should say there is nothing to print")
	}
	if strings.Contains(empty.Body.String(), `class="sheet"`) {
		t.Error("an empty list must not render a blank sheet of paper")
	}
}

// A Dex number is a reasonable thing to type anywhere, and redirecting a download
// to the preview would silently drop the thing the user actually asked for.
func TestSheetsDexRedirectKeepsTheFormat(t *testing.T) {
	rec := get(t, newTestServer(t), "/download/6/sheets?variants=0")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if got, want := rec.Header().Get("Location"), "/download/charizard/sheets?variants=0"; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
}

// An unreachable source must not print as a stack of blank cards.
func TestSheetsSourceUnavailable(t *testing.T) {
	src := testSource(t)
	src.err = errors.New("tcgdex is down")
	rec := get(t, newTestServerWith(t, src), "/download/charizard/sheets")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if strings.Contains(rec.Body.String(), `class="sheet"`) {
		t.Error("a failure must not render printable pages")
	}
}

// blockAt returns the markup from the first occurrence of marker to the end of
// the element that contains it, so an assertion about one placeholder cannot be
// satisfied by text belonging to a different one.
func blockAt(body, marker string) (string, bool) {
	start := strings.Index(body, marker)
	if start < 0 {
		return "", false
	}
	rest := body[start:]
	end := strings.Index(rest, "</div>")
	if end < 0 {
		return rest, true
	}
	return rest[:end], true
}

// inOrder reports whether each string appears after the previous one.
func inOrder(body string, want []string) bool {
	at := 0
	for _, w := range want {
		i := strings.Index(body[at:], w)
		if i < 0 {
			return false
		}
		at += i + len(w)
	}
	return true
}
