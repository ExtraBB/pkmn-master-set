package cards

// The CSV overview: the same list as the preview and the printable sheets, as
// plain rows and columns. It is the escape hatch — the output a collector can
// take away and manage their own way, which is why it carries no formatting, no
// merged cells and no header art, only headers and rows.

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
)

// csvHeader is the column order, taken from docs/variant-taxonomy.md's "Fields on
// every row" — the same fields, in the same order, as the preview and the sheets.
//
// Prices are deliberately absent. The preview shows a current Cardmarket and
// TCGplayer price, but a price baked into a downloaded file goes stale the moment
// it is saved, and this file is meant to be kept.
// The leading "Have" column is the one thing this file carries that the preview
// does not, and it is empty state rather than our state: the product tracks
// nothing, but a spreadsheet the collector owns is exactly where ticking things
// off belongs.
var csvHeader = []string{
	"Have", "Name", "Set", "Number", "Rarity", "Variant", "Language", "Released", "Illustrator",
}

// csvCheckbox is an empty ballot box, U+2610. A CSV cannot carry a real checkbox
// control — that is a spreadsheet feature, not a file format one — so every row
// ships the glyph, which reads as a checkbox on sight and which a collector ticks
// by replacing it with ☑.
const csvCheckbox = "☐"

// variantColumn is the index of the column that changes meaning with the variants
// toggle: the printing's label with variants on, the number of printings the row
// stands for with them off.
const variantColumn = 5

// csvBOM is a UTF-8 byte order mark.
//
// It is here for exactly one reason: Excel on Windows decodes a BOM-less .csv as
// the system's ANSI codepage and mangles every Japanese card name in the file. The
// BOM tells it the truth. Numbers and Google Sheets both honour it and neither
// shows it as content, so the one file opens correctly in all three.
const csvBOM = "\uFEFF"

// WriteCSV writes res as a spreadsheet-ready CSV: a header row, then one row per
// printing in the order res already carries. It never sorts or filters — the whole
// promise of this output is that it is the list the preview showed.
func WriteCSV(w io.Writer, res Result) error {
	if _, err := io.WriteString(w, csvBOM); err != nil {
		return err
	}

	cw := csv.NewWriter(w)
	// CRLF, as RFC 4180 specifies and as Excel is least surprised by. Numbers and
	// Sheets accept either.
	cw.UseCRLF = true

	header := make([]string, len(csvHeader))
	copy(header, csvHeader)
	if !res.IncludeVariants {
		// With variants off a row stands for the card rather than one printing, so
		// the column reports how many printings it folded away. Nothing is deleted
		// by the toggle, and the file says so.
		header[variantColumn] = "Printings"
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, p := range res.Printings {
		if err := cw.Write(csvRow(p, res.IncludeVariants)); err != nil {
			return err
		}
	}

	cw.Flush()
	return cw.Error()
}

// csvRow renders one printing. Every cell goes through Display, so a field the
// source lacks reads as the literal "unknown" here exactly as it does on screen
// and on a placeholder — a blank cell reads as a bug, a stated "unknown" reads as
// the gap it is.
func csvRow(p Printing, includeVariants bool) []string {
	variant := strconv.Itoa(p.Collapsed)
	if includeVariants {
		variant = p.VariantLabel
	}
	return []string{
		csvCheckbox,
		p.Display("name"),
		p.Display("set"),
		p.Display("number"),
		p.Display("rarity"),
		variant,
		p.Display("language"),
		p.Display("year"),
		p.Display("illustrator"),
	}
}

// CSVFilename names the download after the Pokémon and the options it was
// generated under — "charizard-en-variants.csv" — so a folder of downloads stays
// intelligible and two lists for the same Pokémon never collide.
func CSVFilename(res Result) string {
	parts := []string{res.Species.Slug(), string(res.Language)}
	if res.IncludeVariants {
		parts = append(parts, "variants")
	} else {
		parts = append(parts, "no-variants")
	}
	return strings.Join(parts, "-") + ".csv"
}
