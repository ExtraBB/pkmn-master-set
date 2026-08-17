# 004 — PDF and CSV overviews

**Priority:** P1 · **Depends on:** 001

## The need

> As a collector, I want the card list as a document I can read or a spreadsheet I can manage
> myself, so I'm not limited to the binder format.

Two data-led outputs covering the same card list as the printable sheets. Grouped into one issue
because they're the same content in two containers.

## PDF overview — for reading

A text table of the full card list. For reading, filing, or printing as a reference sheet to keep in
the binder's front pocket.

- Text-led, not image-led: set, card number, rarity, variant, language, release date
- Compact — a full Pokémon should fit in a couple of pages
- Same chronological order as the preview and the printable sheets
- Prints legibly on a normal home printer without squinting

## CSV overview — for spreadsheets

One row per card variant, one column per field, plain and unstyled. For collectors who manage their
list their own way — this is the escape hatch that lets someone use the product without accepting
our format.

- No formatting, no merged cells, no header art, no footnotes — just headers and rows
- Opens cleanly in Excel, Numbers and Google Sheets with no manual fixing
- Immediately sortable on every column
- Handles Japanese card names without mangling them

## Acceptance criteria

- Both contain exactly the same card list as the preview, in the same order, with the same fields
- The CSV opens correctly in Excel, Numbers and Google Sheets — verified in all three
- Japanese text survives the round trip into a spreadsheet intact
- The PDF is readable when printed at normal size
- Filenames identify the Pokémon and options used, so a folder of downloads stays intelligible

## Note

The CSV is the lowest-effort, highest-durability output — it's the one that keeps working for a user
no matter what we do later. Worth not over-thinking, and worth not skipping.
