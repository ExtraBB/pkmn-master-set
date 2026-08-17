package cards

import (
	"slices"
	"testing"
)

func v(typ, subtype string, stamps []string, size string) Variant {
	return NormalizeVariant(typ, subtype, stamps, size)
}

// Base Set Charizard is the canonical case: four printings, and a collector
// expects each to be nameable the way they'd say it out loud.
func TestBaseSetCharizardExpandsToFourPrintings(t *testing.T) {
	raw := []Variant{
		{Type: "holo", Subtype: "unlimited", Size: "standard"},
		{Type: "holo", Subtype: "shadowless", Stamps: []string{"1st-edition"}, Size: "standard"},
		{Type: "holo", Subtype: "shadowless", Size: "standard"},
		{Type: "holo", Subtype: "1999-2000-copyright", Size: "standard"},
	}

	got := NormalizeVariants(raw)
	if len(got) != 4 {
		t.Fatalf("got %d printings, want 4", len(got))
	}

	want := []string{
		"1st Edition · Shadowless · Holo",
		"Shadowless · Holo",
		"Unlimited · Holo",
		"1999–2000 Copyright · Holo",
	}
	for i, w := range want {
		if got[i].Label() != w {
			t.Errorf("printing %d = %q, want %q", i, got[i].Label(), w)
		}
	}
}

// The source emits the same variant twice under different internal ids. Printing
// that placeholder twice is exactly the quiet wrongness this product exists to
// avoid, so identity is semantic.
func TestDuplicateVariantsAreDeduped(t *testing.T) {
	raw := []Variant{
		{Type: "holo", Size: "standard"},
		{Type: "reverse", Size: "standard"},
		{Type: "reverse", Size: "standard"},
	}
	if got := NormalizeVariants(raw); len(got) != 2 {
		t.Fatalf("got %d printings, want 2", len(got))
	}
}

// A card with no variant data still occupies a binder slot.
func TestNoVariantsYieldsOnePrinting(t *testing.T) {
	got := NormalizeVariants(nil)
	if len(got) != 1 {
		t.Fatalf("got %d printings, want 1", len(got))
	}
	if got[0].Label() != "Standard" {
		t.Errorf("label = %q, want Standard", got[0].Label())
	}
}

func TestVariantLabels(t *testing.T) {
	tests := []struct {
		name    string
		variant Variant
		label   string
		short   string
	}{
		{"plain non-holo", v("normal", "", nil, "standard"), "Standard", "Standard"},
		{"holo", v("holo", "", nil, "standard"), "Holo", "Holo"},
		{"reverse holo", v("reverse", "", nil, "standard"), "Reverse Holo", "Reverse Holo"},
		{"first edition", v("holo", "", []string{"1st-edition"}, "standard"), "1st Edition · Holo", "1st Edition"},
		{"error print", v("holo", "energy-symbol-error", nil, "standard"), "Energy Symbol Error · Holo", "Energy Symbol Error"},
		{"staff stamp", v("normal", "", []string{"staff"}, "standard"), "Staff Stamp", "Staff Stamp"},
		// The word "Stamp" is appended once, not baked into the table.
		{"set logo stamp", v("holo", "", []string{"set-logo"}, "standard"), "Holo · Set Logo Stamp", "Holo"},
		{"jumbo", v("normal", "", nil, "jumbo"), "Jumbo", "Jumbo"},
		{"japanese no-rarity", v("holo", "no-rarity", nil, "standard"), "No Rarity Symbol · Holo", "No Rarity Symbol"},
		// An unfamiliar stamp must still name itself. Losing a printing because
		// we had not seen its stamp before would be a completeness failure.
		{"unknown stamp", v("normal", "", []string{"worlds-2010"}, "standard"), "Worlds 2010 Stamp", "Worlds 2010 Stamp"},
		{"unknown subtype", v("holo", "cosmos-foil", nil, "standard"), "Cosmos Foil · Holo", "Cosmos Foil"},
		{"unknown finish", v("cracked-ice", "", nil, "standard"), "Cracked Ice", "Cracked Ice"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.variant.Label(); got != tc.label {
				t.Errorf("Label() = %q, want %q", got, tc.label)
			}
			if got := tc.variant.Short(); got != tc.short {
				t.Errorf("Short() = %q, want %q", got, tc.short)
			}
		})
	}
}

func TestVariantKeyIsStableAndOrderIndependent(t *testing.T) {
	a := NormalizeVariant("HOLO", "Shadowless", []string{"staff", "1st-edition"}, "STANDARD")
	b := NormalizeVariant("holo", "shadowless", []string{"1st-edition", "staff"}, "standard")
	if a.Key() != b.Key() {
		t.Errorf("keys differ: %q vs %q", a.Key(), b.Key())
	}
}

// Turning variants off should land on the card a collector pictures when they
// say "I need the Base Set Charizard" — not a stamped or 1st Edition copy.
func TestCollapsePrefersThePlainestPrinting(t *testing.T) {
	tests := []struct {
		name string
		in   []Variant
		want string
	}{
		{
			"prefers unstamped non-holo",
			NormalizeVariants([]Variant{
				{Type: "holo", Stamps: []string{"staff"}},
				{Type: "normal"},
				{Type: "reverse"},
			}),
			"normal|||standard",
		},
		{
			"falls back to holo when there is no non-holo",
			NormalizeVariants([]Variant{
				{Type: "holo", Subtype: "shadowless", Stamps: []string{"1st-edition"}},
				{Type: "holo", Subtype: "unlimited"},
			}),
			"holo|unlimited||standard",
		},
		{
			// Some cards only ever existed stamped. There is no plain printing to
			// fall back to, so the earliest one stands in rather than nothing.
			"falls back to the first printing when all are stamped",
			NormalizeVariants([]Variant{
				{Type: "holo", Stamps: []string{"staff"}},
				{Type: "holo", Stamps: []string{"prerelease"}},
			}),
			"holo||prerelease|standard",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Collapse(tc.in).Key(); got != tc.want {
				t.Errorf("Collapse = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestVariantOrderIsChronological(t *testing.T) {
	got := NormalizeVariants([]Variant{
		{Type: "reverse"},
		{Type: "holo", Subtype: "unlimited"},
		{Type: "holo", Subtype: "shadowless"},
		{Type: "holo", Subtype: "shadowless", Stamps: []string{"1st-edition"}},
		{Type: "normal"},
	})

	labels := make([]string, len(got))
	for i, g := range got {
		labels[i] = g.Label()
	}
	want := []string{
		"1st Edition · Shadowless · Holo",
		"Shadowless · Holo",
		"Standard",
		"Unlimited · Holo",
		"Reverse Holo",
	}
	if !slices.Equal(labels, want) {
		t.Errorf("order =\n %v\nwant\n %v", labels, want)
	}
}
