# 004 — CSV overview

**Priority:** P1 · **Depends on:** 001

## The need

> As a collector, I want the card list as a spreadsheet I can manage myself, so I'm not limited to
> the binder format.

A data-led output covering the same card list as the printable sheets.

## CSV overview — for spreadsheets

One row per card variant, one column per field, plain and unstyled. For collectors who manage their
list their own way — this is the escape hatch that lets someone use the product without accepting
our format.

- No formatting, no merged cells, no header art, no footnotes — just headers and rows
- Opens cleanly in Excel, Numbers and Google Sheets with no manual fixing
- Immediately sortable on every column
- Handles Japanese card names without mangling them
- Same fields and same chronological order as the preview and the printable sheets

## Acceptance criteria

- It contains exactly the same card list as the preview, in the same order, with the same fields
- The CSV opens correctly in Excel, Numbers and Google Sheets — verified in all three
- Japanese text survives the round trip into a spreadsheet intact
- The filename identifies the Pokémon and options used, so a folder of downloads stays intelligible

## Note

The CSV is the lowest-effort, highest-durability output — it's the one that keeps working for a user
no matter what we do later. Worth not over-thinking, and worth not skipping.
