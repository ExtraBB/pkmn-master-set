package cards

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"
)

// writeCSV runs a query through WriteCSV and returns the raw bytes.
func writeCSV(t *testing.T, src *fakeSource, q Query) string {
	t.Helper()
	res, err := testCatalog(t, src).Query(context.Background(), q)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	var buf bytes.Buffer
	if err := WriteCSV(&buf, res); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	return buf.String()
}

// parseCSV strips the BOM and reads the file back the way a spreadsheet would.
func parseCSV(t *testing.T, out string) [][]string {
	t.Helper()
	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(out, csvBOM))).ReadAll()
	if err != nil {
		t.Fatalf("re-reading our own CSV: %v", err)
	}
	return rows
}

// The CSV is the same list as the preview: same rows, same order, same fields —
// and a field the source lacks says "unknown" rather than leaving a blank cell.
func TestWriteCSV(t *testing.T) {
	out := writeCSV(t, charizardFixture(t), Query{DexID: 6, Language: LangEN, IncludeVariants: true})
	rows := parseCSV(t, out)

	want := [][]string{
		{"Have", "Name", "Set", "Number", "Rarity", "Variant", "Language", "Released", "Illustrator"},
		{"☐", "Charizard", "Base Set", "4/102", "Rare Holo", "1st Edition · Shadowless · Holo", "English", "1999", "Mitsuhiro Arita"},
		{"☐", "Charizard", "Base Set", "4/102", "Rare Holo", "Shadowless · Holo", "English", "1999", "Mitsuhiro Arita"},
		{"☐", "Charizard", "Base Set", "4/102", "Rare Holo", "Unlimited · Holo", "English", "1999", "Mitsuhiro Arita"},
		{"☐", "Charizard", "Base Set 2", "4/130", "Rare Holo", "Holo", "English", "2000", "Mitsuhiro Arita"},
		// The awkward card: no rarity, no illustrator, an undated set. Every gap
		// reads as "unknown", and it sorts to the end because its date is unknown.
		{"☐", "Charizard", "McDonald's Collection 2024", "1/15", "unknown", "Standard", "English", "unknown", "unknown"},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d:\n%q", len(rows), len(want), out)
	}
	for i, w := range want {
		if got := rows[i]; !equalRow(got, w) {
			t.Errorf("row %d = %q\n          want %q", i, got, w)
		}
	}
}

func equalRow(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Turning variants off folds rows away rather than deleting cards, so the column
// that held the printing's label reports how many printings the row stands for.
func TestWriteCSVVariantsOff(t *testing.T) {
	out := writeCSV(t, charizardFixture(t), Query{DexID: 6, Language: LangEN})
	rows := parseCSV(t, out)

	if got := rows[0][variantColumn]; got != "Printings" {
		t.Errorf("header column %d = %q, want %q", variantColumn, got, "Printings")
	}
	if len(rows) != 4 { // three cards, plus the header
		t.Fatalf("got %d rows, want 4:\n%q", len(rows), out)
	}
	// Base Set Charizard has three printings; the other two cards have one each.
	for i, want := range []string{"3", "1", "1"} {
		if got := rows[i+1][variantColumn]; got != want {
			t.Errorf("row %d printings = %q, want %q", i+1, got, want)
		}
	}
}

// The checkbox column leads every row and starts empty on all of them. It is the
// collector's state to fill in, in their own file — the product itself tracks
// nothing and must never ship a row pre-ticked.
func TestWriteCSVCheckboxColumn(t *testing.T) {
	out := writeCSV(t, charizardFixture(t), Query{DexID: 6, Language: LangEN, IncludeVariants: true})
	rows := parseCSV(t, out)

	if got := rows[0][0]; got != "Have" {
		t.Errorf("header column 0 = %q, want %q", got, "Have")
	}
	for i, r := range rows[1:] {
		if r[0] != csvCheckbox {
			t.Errorf("row %d checkbox = %q, want an empty box %q", i+1, r[0], csvCheckbox)
		}
	}
}

// Excel on Windows decodes a BOM-less .csv as the system codepage and mangles
// every Japanese name in it, so the BOM is not decoration — it is the mechanism
// behind "opens correctly in Excel, Numbers and Google Sheets".
func TestWriteCSVIsSpreadsheetShaped(t *testing.T) {
	out := writeCSV(t, charizardFixture(t), Query{DexID: 6, Language: LangEN, IncludeVariants: true})

	if !strings.HasPrefix(out, csvBOM) {
		t.Errorf("CSV does not start with a UTF-8 BOM: %q", out[:min(12, len(out))])
	}
	if !strings.Contains(out, "\r\n") {
		t.Error("CSV does not use CRLF line endings")
	}
	// Every row the same width, or a spreadsheet cannot sort a column.
	rows := parseCSV(t, out)
	for i, r := range rows {
		if len(r) != len(csvHeader) {
			t.Errorf("row %d has %d fields, want %d", i, len(r), len(csvHeader))
		}
	}
}

// Japanese card names have to survive the round trip into a spreadsheet intact.
func TestWriteCSVJapanese(t *testing.T) {
	src := &fakeSource{
		sets: map[string]Set{"base1": {ID: "base1", Name: "拡張パック", CountOfficial: 102}},
		cards: map[Language][]Card{
			LangJA: {{
				ID: "base1-4", Name: "リザードン", Number: "4", DexIDs: []int{6},
				Rarity: "レア", Illustrator: "さいとうなおき",
				SetID: "base1", SetName: "拡張パック", SetTotal: 102,
				Variants: []Variant{{Type: "holo", Size: "standard"}},
			}},
		},
	}
	out := writeCSV(t, src, Query{DexID: 6, Language: LangJA, IncludeVariants: true})

	rows := parseCSV(t, out)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2:\n%q", len(rows), out)
	}
	// Offset by one: the checkbox column leads every row.
	for i, want := range []string{"リザードン", "拡張パック", "4/102", "レア"} {
		if got := rows[1][i+1]; got != want {
			t.Errorf("field %d = %q, want %q", i+1, got, want)
		}
	}
	if got := rows[1][6]; got != "Japanese" {
		t.Errorf("language = %q, want %q", got, "Japanese")
	}
}

// A Pokémon with no cards in the chosen language still yields a valid, openable
// file — a header and nothing else, not an empty download.
func TestWriteCSVEmpty(t *testing.T) {
	out := writeCSV(t, charizardFixture(t), Query{DexID: 25, Language: LangEN, IncludeVariants: true})
	rows := parseCSV(t, out)
	if len(rows) != 1 || rows[0][0] != "Have" {
		t.Errorf("empty result = %q, want the header row alone", rows)
	}
}

// The filename has to identify the Pokémon and the options, so a folder of
// downloads stays intelligible and two lists never collide.
func TestCSVFilename(t *testing.T) {
	tests := []struct {
		res  Result
		want string
	}{
		{Result{Species: Species{Name: "Charizard"}, Language: LangEN, IncludeVariants: true}, "charizard-en-variants.csv"},
		{Result{Species: Species{Name: "Charizard"}, Language: LangEN}, "charizard-en-no-variants.csv"},
		{Result{Species: Species{Name: "Charizard"}, Language: LangJA, IncludeVariants: true}, "charizard-ja-variants.csv"},
		// A name with punctuation still yields a filename safe on every filesystem.
		{Result{Species: Species{Name: "Mr. Mime"}, Language: LangEN, IncludeVariants: true}, "mr-mime-en-variants.csv"},
		{Result{Species: Species{Name: "Nidoran♀"}, Language: LangEN, IncludeVariants: true}, "nidoran-f-en-variants.csv"},
	}
	for _, tt := range tests {
		if got := CSVFilename(tt.res); got != tt.want {
			t.Errorf("CSVFilename(%s, %s, variants=%v) = %q, want %q",
				tt.res.Species.Name, tt.res.Language, tt.res.IncludeVariants, got, tt.want)
		}
	}
}
