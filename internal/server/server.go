// Package server renders the whole web product: a search box, and a preview of
// the card list a collector is about to download.
//
// The preview is a confirmation step, not a workspace — there is deliberately no
// sorting, filtering, per-card selection or ownership state anywhere below.
package server

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ExtraBB/pkmn-master-set/internal/cards"
	"github.com/ExtraBB/pkmn-master-set/web"
)

// rowsPerRequest is how many rows the preview shows before asking whether you want
// more. The full list stays reachable — a preview that hides most of the list
// cannot answer the completeness question — but a few hundred thumbnails at once is
// a slow first paint for a decision the first batch already supports.
const rowsPerRequest = 50

// queryTimeout bounds one card-list fetch. Every list costs one upstream request
// per card in every language, so this is generous rather than tight.
const queryTimeout = 30 * time.Second

// suggestionLimit is how many typeahead rows to offer: enough that "char" shows the
// whole Charmander line, few enough to stay a glance.
const suggestionLimit = 8

// pageFiles are the full-page templates. Each is parsed into its own set alongside
// the layout and the partials, so every page can define "main" without colliding
// with the others.
var pageFiles = []string{"landing.html", "preview.html", "notfound.html", "unavailable.html"}

// Server holds everything the HTTP handlers need.
type Server struct {
	catalog *cards.Catalog
	pages   map[string]*template.Template
	// partials renders the htmx fragments, which belong to no single page.
	partials *template.Template
	// sheets is the printable sheet, parsed alone. It deliberately does not use
	// the site layout: a sheet is a physical artifact, and the header, the
	// wordmark, the fonts and the dark palette are all things you do not want
	// anywhere near a page you are about to cut up.
	sheets *template.Template
}

// New builds the server around a card catalog. The catalog is the product: every
// page and every download is a rendering of what it returns.
func New(catalog *cards.Catalog) *Server {
	s := &Server{catalog: catalog, pages: make(map[string]*template.Template, len(pageFiles))}
	for _, p := range pageFiles {
		s.pages[p] = template.Must(template.New(p).Funcs(funcs).ParseFS(web.Templates,
			"templates/layout.html", "templates/partials/*.html", "templates/pages/"+p))
	}
	s.partials = template.Must(template.New("partials").Funcs(funcs).ParseFS(web.Templates,
		"templates/partials/*.html"))
	s.sheets = template.Must(template.New("sheets.html").Funcs(funcs).ParseFS(web.Templates,
		"templates/pages/sheets.html"))
	return s
}

// funcs are the template helpers. Only one: this screen is all about counts, and
// "1 pages" undermines the numbers it is trying to make trustworthy.
var funcs = template.FuncMap{
	"plural": func(n int, singular, plural string) string {
		if n == 1 {
			return singular
		}
		return plural
	},
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", cacheStatic(http.FileServerFS(web.Static)))
	mux.HandleFunc("GET /{$}", s.handleLanding)
	mux.HandleFunc("GET /search", s.handleSearch)
	mux.HandleFunc("GET /go", s.handleGo)

	// A Pokémon owns a bare, shareable URL — /charizard. That is only unambiguous
	// while the root holds single-segment names, so the fragment and download
	// routes live under prefixes of their own rather than under the Pokémon.
	mux.HandleFunc("GET /rows/{slug}", s.handleRows)
	mux.HandleFunc("GET /download/{slug}/{format}", s.handleDownload)
	mux.HandleFunc("GET /{slug}", s.handlePreview)

	return mux
}

// cacheStatic lets browsers keep the CSS, htmx and the 1,025 Dex sprites, which are
// immutable content addressed by Dex number.
func cacheStatic(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		h.ServeHTTP(w, r)
	})
}

// ---- landing and search ----

// landingData backs the landing page and the not-found page: a search box, four
// examples, and — when a lookup failed — the name that was typed, so the message
// can name it back instead of saying "not found" into the void.
type landingData struct {
	Query    string
	Examples []cards.Species
}

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, "landing.html", http.StatusOK, landingData{Examples: s.examples()})
}

// examples are the four Pokémon offered under the search box. They are looked up
// rather than written out, so a link can never point at a Pokémon the index lacks.
func (s *Server) examples() []cards.Species {
	out := make([]cards.Species, 0, 4)
	for _, dexID := range []int{6, 25, 133, 94} { // Charizard, Pikachu, Eevee, Gengar
		if sp, ok := s.catalog.Species(dexID); ok {
			out = append(out, sp)
		}
	}
	return out
}

type suggestionsData struct {
	Query   string
	Results []cards.Species
}

// handleSearch answers the typeahead with an HTML fragment.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	data := suggestionsData{Query: q}
	if q != "" {
		data.Results = s.catalog.Search(q, suggestionLimit)
	}
	s.renderPartial(w, "suggestions", data)
}

// handleGo resolves a submitted search box, so pressing Enter works without picking
// a suggestion — and without JavaScript at all.
func (s *Server) handleGo(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if sp, ok := s.catalog.SpeciesByName(q); ok {
		http.Redirect(w, r, "/"+sp.Slug(), http.StatusSeeOther)
		return
	}
	// No exact name: take the best typeahead match, which is what the user was
	// looking at when they pressed Enter.
	if hits := s.catalog.Search(q, 1); len(hits) == 1 {
		http.Redirect(w, r, "/"+hits[0].Slug(), http.StatusSeeOther)
		return
	}
	s.notFound(w, q)
}

func (s *Server) notFound(w http.ResponseWriter, query string) {
	s.renderPage(w, "notfound.html", http.StatusNotFound, landingData{
		Query:    query,
		Examples: s.examples(),
	})
}

// ---- preview ----

// previewData is everything the preview screen renders. It carries the whole result
// so the counts, the rows and the download sizes are read from one answer and
// cannot disagree with each other.
type previewData struct {
	Result cards.Result
	Rows   []cards.Printing
	Offset int
	// Host is the host shown in the shareable-URL affordance.
	Host string
}

func (d previewData) Species() cards.Species { return d.Result.Species }
func (d previewData) Slug() string           { return d.Result.Species.Slug() }

// Shown is how many rows are on screen; Remaining is how many more the file holds.
func (d previewData) Shown() int     { return d.Offset + len(d.Rows) }
func (d previewData) Remaining() int { return d.Result.PrintingCount - d.Shown() }
func (d previewData) HasMore() bool  { return d.Remaining() > 0 }

// NextBatch is how many more rows the "show more" control will reveal.
func (d previewData) NextBatch() int { return min(d.Remaining(), rowsPerRequest) }

// URL builds this preview's URL under a different set of options, which is how the
// option controls work: every toggle is a real link to a real shareable URL.
func (d previewData) URL(lang cards.Language, includeVariants bool) string {
	u := "/" + d.Slug()
	if q := options(lang, includeVariants); len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

// Canonical is this exact list's shareable URL.
func (d previewData) Canonical() string { return d.URL(d.Result.Language, d.Result.IncludeVariants) }

// The option controls are named URLs rather than one URL method taking arguments,
// because a template cannot hand a plain string to a cards.Language parameter.
func (d previewData) EnglishURL() string  { return d.URL(cards.LangEN, d.Result.IncludeVariants) }
func (d previewData) JapaneseURL() string { return d.URL(cards.LangJA, d.Result.IncludeVariants) }
func (d previewData) VariantsOnURL() string {
	return d.URL(d.Result.Language, true)
}
func (d previewData) VariantsOffURL() string { return d.URL(d.Result.Language, false) }

func (d previewData) IsJapanese() bool { return d.Result.Language == cards.LangJA }

// RowsURL is the fragment URL for the next batch of rows.
func (d previewData) RowsURL() string {
	q := options(d.Result.Language, d.Result.IncludeVariants)
	q.Set("offset", strconv.Itoa(d.Shown()))
	return "/rows/" + d.Slug() + "?" + q.Encode()
}

// DownloadURL is one format's download link, carrying the same options, so the file
// you get is the list you were looking at.
func (d previewData) DownloadURL(format string) string {
	u := "/download/" + d.Slug() + "/" + format
	if q := options(d.Result.Language, d.Result.IncludeVariants); len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

// queryString preserves the chosen options across a redirect, so a link carrying
// ?variants=0 still lands on the list it promised.
func queryString(r *http.Request) string {
	if raw := r.URL.RawQuery; raw != "" {
		return "?" + raw
	}
	return ""
}

// redirectTarget rewrites this request's own path with the canonical slug, so
// /6 goes to /charizard and /download/6/sheets goes to /download/charizard/sheets.
// Redirecting a download to the preview instead would silently drop the thing the
// user actually asked for.
func redirectTarget(r *http.Request, slug string) string {
	segments := strings.Split(r.URL.Path, "/")
	for i, seg := range segments {
		if seg == r.PathValue("slug") {
			segments[i] = slug
			break
		}
	}
	return strings.Join(segments, "/") + queryString(r)
}

// options encodes only what differs from the defaults, so the common URL stays the
// bare /charizard the design shows.
func options(lang cards.Language, includeVariants bool) url.Values {
	q := url.Values{}
	if lang != cards.LangEN {
		q.Set("lang", string(lang))
	}
	if !includeVariants {
		q.Set("variants", "0")
	}
	return q
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	data, ok := s.preview(w, r, 0)
	if !ok {
		return
	}
	// htmx swaps the panel in place when an option is toggled; a plain request gets
	// the whole page. One handler, one template, one shareable URL.
	if r.Header.Get("HX-Request") != "" {
		s.renderPartial(w, "preview-panel", data)
		return
	}
	s.renderPage(w, "preview.html", http.StatusOK, data)
}

// handleRows returns the next batch of rows plus a refreshed footer, so the whole
// list is reachable without ever rendering all of it at once.
func (s *Server) handleRows(w http.ResponseWriter, r *http.Request) {
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}
	data, ok := s.preview(w, r, offset)
	if !ok {
		return
	}
	s.renderPartial(w, "rows-batch", data)
}

// preview resolves the slug and the options, runs the query and slices out one
// batch of rows. It writes the error page itself and reports whether the caller
// should carry on.
func (s *Server) preview(w http.ResponseWriter, r *http.Request, offset int) (previewData, bool) {
	res, ok := s.resolve(w, r)
	if !ok {
		return previewData{}, false
	}

	rows := res.Printings
	if offset > len(rows) {
		offset = len(rows)
	}
	rows = rows[offset:]
	if len(rows) > rowsPerRequest {
		rows = rows[:rowsPerRequest]
	}
	return previewData{Result: res, Rows: rows, Offset: offset, Host: r.Host}, true
}

// resolve turns a request into one answer from the catalog: the slug, the
// options, the query. It writes the not-found or unavailable page itself and
// reports whether the caller should carry on.
//
// The preview and the printable sheet both start here, so the list you print is
// by construction the list you were shown.
func (s *Server) resolve(w http.ResponseWriter, r *http.Request) (cards.Result, bool) {
	slug := r.PathValue("slug")
	sp, ok := s.catalog.SpeciesByName(slug)
	if !ok {
		// A Dex number in the path is a reasonable thing to type, but a Pokémon has
		// one canonical URL, so redirect to it rather than serving two.
		if dexID, err := strconv.Atoi(slug); err == nil {
			if sp, found := s.catalog.Species(dexID); found {
				http.Redirect(w, r, redirectTarget(r, sp.Slug()), http.StatusSeeOther)
				return cards.Result{}, false
			}
		}
		s.notFound(w, slug)
		return cards.Result{}, false
	}

	q := cards.Query{
		DexID:           sp.DexID,
		Language:        language(r.URL.Query().Get("lang")),
		IncludeVariants: includeVariants(r.URL.Query().Get("variants")),
	}

	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()

	res, err := s.catalog.Query(ctx, q)
	if err != nil {
		if errors.Is(err, cards.ErrUnknownSpecies) {
			s.notFound(w, slug)
			return cards.Result{}, false
		}
		// Anything else is the source being unreachable, and saying so is the only
		// honest option: an empty table would read as "this Pokémon has no cards".
		log.Printf("query %s (%s): %v", sp.Name, q.Language, err)
		s.renderPage(w, "unavailable.html", http.StatusServiceUnavailable, previewData{
			Result: cards.Result{Species: sp, Language: q.Language, IncludeVariants: q.IncludeVariants},
			Host:   r.Host,
		})
		return cards.Result{}, false
	}
	return res, true
}

// handleDownload serves one output format. Sheets are here; the CSV overview
// arrives with issue 004.
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	switch format := r.PathValue("format"); format {
	case "sheets":
		s.handleSheets(w, r)
	case "csv":
		http.Error(w, "not implemented yet: "+format+" downloads arrive with issue 004",
			http.StatusNotImplemented)
	default:
		http.NotFound(w, r)
	}
}

// ---- printable sheets ----

// handleSheets renders the core output: card-sized placeholders at true 63x88mm,
// nine to a page, in the same order the preview showed.
//
// It is a page rather than a file. The artifact the collector wants is ink on
// paper, and the browser's own print pipeline is the only thing that can put it
// there at an exact physical size — so the product hands over a page built for
// printing, with the scaling warning on it, instead of a file that would have to
// be printed by the same browser anyway.
func (s *Server) handleSheets(w http.ResponseWriter, r *http.Request) {
	res, ok := s.resolve(w, r)
	if !ok {
		return
	}
	data := sheetsData{Result: res, Art: artMode(r.URL.Query().Get("art"))}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.sheets.ExecuteTemplate(w, "sheets.html", data); err != nil {
		log.Printf("render sheets %s: %v", res.Species.Name, err)
	}
}

// Art treatments. A placeholder marks a gap in a binder, so the default greys the
// art out: a gap should look like a gap when you flip past it, and a greyed
// placeholder is not a full-quality reproduction of a card at card size. Full
// colour and a text-only outline are both a click away for collectors who want
// recognisability or want to spare the ink.
const (
	artGrey    = "grey"
	artColour  = "colour"
	artOutline = "outline"
)

// artMode falls back to the default rather than failing, matching language() and
// includeVariants(): a mistyped parameter in a shared link should still print.
func artMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case artColour:
		return artColour
	case artOutline:
		return artOutline
	default:
		return artGrey
	}
}

// sheetsData backs the printable sheet.
type sheetsData struct {
	Result cards.Result
	Art    string
}

func (d sheetsData) Species() cards.Species { return d.Result.Species }
func (d sheetsData) Slug() string           { return d.Result.Species.Slug() }

func (d sheetsData) IsJapanese() bool { return d.Result.Language == cards.LangJA }

func (d sheetsData) IsColour() bool  { return d.Art == artColour }
func (d sheetsData) IsOutline() bool { return d.Art == artOutline }
func (d sheetsData) IsGrey() bool    { return d.Art == artGrey }

// Pages chunks the list into sheets of paper. It is done here rather than in the
// template so the number of placeholders on a page and the page count the preview
// promised come from the same constant and cannot drift apart.
func (d sheetsData) Pages() [][]cards.Printing {
	per := cards.PlaceholdersPerPage
	all := d.Result.Printings
	out := make([][]cards.Printing, 0, cards.SheetPages(len(all)))
	for start := 0; start < len(all); start += per {
		out = append(out, all[start:min(start+per, len(all))])
	}
	return out
}

// BackURL is the preview this sheet came from, carrying the same options.
func (d sheetsData) BackURL() string {
	u := "/" + d.Slug()
	if q := options(d.Result.Language, d.Result.IncludeVariants); len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

// ArtURL is this same sheet under a different art treatment — a real link, so
// every version of the sheet has a URL you can bookmark or share.
func (d sheetsData) ArtURL(art string) string {
	q := options(d.Result.Language, d.Result.IncludeVariants)
	if art != artGrey {
		q.Set("art", art)
	}
	u := "/download/" + d.Slug() + "/sheets"
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

// Named URLs, because a template cannot hand a plain string to a typed constant.
func (d sheetsData) GreyURL() string    { return d.ArtURL(artGrey) }
func (d sheetsData) ColourURL() string  { return d.ArtURL(artColour) }
func (d sheetsData) OutlineURL() string { return d.ArtURL(artOutline) }

// language falls back to English rather than failing: a mistyped parameter in a
// shared link should still show a list.
func language(v string) cards.Language {
	if l := cards.Language(strings.ToLower(strings.TrimSpace(v))); l.Valid() {
		return l
	}
	return cards.LangEN
}

// includeVariants defaults to on, because a printing per placeholder is the
// product's stated unit.
func includeVariants(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

// ---- rendering ----

func (s *Server) renderPage(w http.ResponseWriter, page string, status int, data any) {
	tmpl, ok := s.pages[page]
	if !ok {
		log.Printf("render: unknown page %q", page)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		// The status line has already gone out, so recording it is all that is left.
		log.Printf("render %s: %v", page, err)
	}
}

func (s *Server) renderPartial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.partials.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render partial %s: %v", name, err)
	}
}
