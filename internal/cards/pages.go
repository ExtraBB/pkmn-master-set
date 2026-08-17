package cards

// Page arithmetic lives in the domain because the preview promises a number the
// downloads then have to honour: "214 cards · ~24 pages" is shown before anyone
// commits paper, so the estimate and the generator must never drift apart.

// PlaceholdersPerPage is how many true-size placeholders fit on one sheet.
//
// A 3x3 grid of 63x88mm cards occupies 189x264mm, which fits inside both A4
// (210x297) and US Letter (216x279) with room for margins and cut lines. Because
// both paper sizes take the same grid, the page count shown in the preview is
// correct whichever the user prints on.
const PlaceholdersPerPage = 9

// SheetPages is how many printable sheets n placeholders take.
func SheetPages(n int) int { return ceilDiv(n, PlaceholdersPerPage) }

// ceilDiv rounds up. An empty list is zero pages, not one: there is nothing to
// print, and claiming a page would be a lie the user pays for in paper.
func ceilDiv(n, per int) int {
	if n <= 0 {
		return 0
	}
	return (n + per - 1) / per
}

// SheetPages is the paper cost of this result — the number the user is really
// deciding about on the preview screen.
func (r Result) SheetPages() int { return SheetPages(r.PrintingCount) }

// HasFoldedPrintings reports whether turning variants off folded any rows away.
// When it is true the preview can state the delta, so the effect of the toggle is
// visible rather than discovered after printing.
func (r Result) HasFoldedPrintings() bool { return r.VariantCount > r.PrintingCount }

// FoldedPrintings is how many rows the variants toggle is currently hiding.
func (r Result) FoldedPrintings() int {
	if !r.HasFoldedPrintings() {
		return 0
	}
	return r.VariantCount - r.PrintingCount
}

// FoldedPages is how many sheets of paper the variants toggle is currently saving.
func (r Result) FoldedPages() int {
	if !r.HasFoldedPrintings() {
		return 0
	}
	return SheetPages(r.VariantCount) - r.SheetPages()
}
