package cards

import (
	"slices"
	"strings"
)

// SortPrintings puts a card list in the order the product promises everywhere:
// chronological by set release, so a filled binder reads as the Pokémon's history
// from front to back. The preview, the CSV and the printed sheets all
// use this single ordering — "what you see is the order you get".
func SortPrintings(ps []Printing) {
	slices.SortStableFunc(ps, comparePrinting)
}

func comparePrinting(a, b Printing) int {
	// 1. Release date. Undated sets sort last rather than first: a dateless promo
	//    set at the front of the binder would corrupt the chronology, and "we
	//    don't know when this came out" belongs at the end.
	if c := compareReleaseDate(a.ReleaseDate, b.ReleaseDate); c != 0 {
		return c
	}
	// 2. Sets released the same day need a stable tiebreak.
	if c := strings.Compare(a.SetName, b.SetName); c != 0 {
		return c
	}
	if c := strings.Compare(a.SetID, b.SetID); c != 0 {
		return c
	}
	// 3. Card number, compared naturally so 9 precedes 10.
	if c := CompareCardNumber(a.Number, b.Number); c != 0 {
		return c
	}
	if c := strings.Compare(a.CardID, b.CardID); c != 0 {
		return c
	}
	// 4. Printings of the same card, earliest print run first.
	return compareVariant(a.Variant, b.Variant)
}

func compareReleaseDate(a, b Date) int {
	switch {
	case a.IsZero() && b.IsZero():
		return 0
	case a.IsZero():
		return 1
	case b.IsZero():
		return -1
	case a.Before(b):
		return -1
	case b.Before(a):
		return 1
	default:
		return 0
	}
}

// CompareCardNumber orders card numbers the way they are printed on the card
// rather than the way strings sort.
//
// Plain string comparison puts "10" before "9" and "TG12" before "TG5", which
// would shuffle the printed sheet out of card order. Splitting into digit and
// non-digit runs and comparing run by run handles the real vocabulary of card
// numbers: "9" < "10", "4" < "4a", "TG05" < "TG12", "H12", "SV107".
func CompareCardNumber(a, b string) int {
	ai, bi := 0, 0
	for ai < len(a) && bi < len(b) {
		aRun, aNext, aDigits := nextRun(a, ai)
		bRun, bNext, bDigits := nextRun(b, bi)

		switch {
		case aDigits && bDigits:
			// Compare numerically; leading zeros ("05" vs "5") tie-break on the
			// literal text so the ordering stays total.
			if c := compareNumericRun(aRun, bRun); c != 0 {
				return c
			}
			if c := strings.Compare(aRun, bRun); c != 0 {
				return c
			}
		case aDigits != bDigits:
			// A number sorts before a letter at the same position: "4" < "4a"
			// falls out of the shorter string ending first, but "12" < "H12"
			// needs this rule.
			if aDigits {
				return -1
			}
			return 1
		default:
			if c := strings.Compare(strings.ToLower(aRun), strings.ToLower(bRun)); c != 0 {
				return c
			}
		}
		ai, bi = aNext, bNext
	}

	switch {
	case ai < len(a):
		return 1
	case bi < len(b):
		return -1
	default:
		return 0
	}
}

// nextRun returns the run of same-class characters starting at i, the index just
// past it, and whether the run is digits.
func nextRun(s string, i int) (run string, next int, digits bool) {
	digits = isDigit(s[i])
	j := i
	for j < len(s) && isDigit(s[j]) == digits {
		j++
	}
	return s[i:j], j, digits
}

// compareNumericRun compares two digit runs by value without converting them, so
// arbitrarily long runs cannot overflow.
func compareNumericRun(a, b string) int {
	a = strings.TrimLeft(a, "0")
	b = strings.TrimLeft(b, "0")
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
