# PRD — pkmn-master-set

*Draft v2 · 2026-08-17 · product scope only (no technical design)*

## Context

Collectors chasing every card of a single Pokémon track their progress **physically** — in a binder,
on paper. They don't want another app with another login and another list to keep in sync with
reality. The binder *is* the source of truth.

What they lack is the starting artifact: a complete, accurate, printable list of what exists.
Assembling one today means trawling set-by-set databases and hand-building a spreadsheet.

This product does one thing: **you pick a Pokémon, you see what you're about to get, you download
something you can print.** No accounts, no digital ownership state, no syncing. The output goes to a
printer or a spreadsheet and the product's job is finished.

**Intended outcome:** in under a minute, a collector has a printed set of placeholder cards sitting
in their binder showing exactly which cards they still need to find.

## The whole product

```
   ┌──────────┐     ┌──────────────┐     ┌─────────────┐     ┌────────────┐
   │ Pick a   │  →  │ Preview the  │  →  │ Pick a      │  →  │ File       │
   │ Pokémon  │     │ card list    │     │ format      │     │ downloaded │
   └──────────┘     └──────────────┘     └─────────────┘     └────────────┘
```

Search, look, download. The preview is a confirmation step, not a workspace — keeping the flow this
short is a product requirement, not an aspiration.

## Users

- **Primary — the binder collector.** Collects one Pokémon, tracks it physically, wants placeholder
  cards for the gaps so the binder reads as a complete set-in-progress.
- **Secondary — the spreadsheet collector.** Wants the raw list to manage in their own sheet, their
  own way. Served entirely by the CSV.

## Explicitly not this product

| Not doing | Why |
| --- | --- |
| Digital ownership tracking, checkboxes, progress % | Tracking is physical. This is the pivot: we generate the artifact, the binder holds the state. |
| Accounts, logins, saved collections | Nothing to save. The output is a file. |
| Prices | The product is about what exists, not what it costs. |
| Gameplay data (attacks, HP, abilities) | Not what a collector needs on a placeholder or a checklist. |
| A browsing/filtering workspace | The preview exists to confirm what you're downloading, not to become a place you spend time. The download is the deliverable. |

## The three outputs

All three cover the same card list for the chosen Pokémon. They differ in what they're *for*.

### 1. Printable sheets — the core output
Card-sized placeholders, printed and cut out, slotted into empty binder pockets.

```
A4 / Letter page — cards at true size, cut lines between
┌───────┬───────┬───────┐
│ [img] │ [img] │ [img] │  ← cut lines
│Charz  │Charz  │Charz  │
│4/102  │4/102  │4/102  │
│Unltd  │Shdwls │1st Ed │
├───────┼───────┼───────┤
│ [img] │ [img] │ [img] │
│Base2  │Base2  │Rocket │
└───────┴───────┴───────┘
```

- Cards render at **true trading-card size (63 × 88 mm)** so a cut-out placeholder sits correctly in
  a standard binder pocket. This is the single most important quality bar for this output.
- Image-led: the card art is the identifier. A collector should recognise the missing card at a
  glance while flipping the binder.
- Each placeholder carries a minimal caption: set, card number, variant.
- Cut lines between cards, and as many cards per page as fit at true size.
- Ordered chronologically by set release, so the binder tells the Pokémon's history front to back.

**The known failure mode:** browser and printer "fit to page" / "shrink to fit" scaling silently
resizes the page, and the cut-outs come out wrong and don't fit the pockets. The product must
actively defend against this — clear print instructions, and ideally a printed measurement ruler or
reference mark on the page so the user can verify scale before cutting a whole stack.

### 2. PDF overview — for reading
A text table of the full card list. For reading, filing, or printing as a reference sheet to keep in
the binder's front pocket. Text-led, not image-led: set, card number, rarity, variant, language,
release date. Compact enough that a full Pokémon fits in a couple of pages.

### 3. CSV overview — for spreadsheets
One row per card variant, one column per field, plain and unstyled. For collectors who want to
manage the list themselves. No formatting, no merged cells, no header art — it must open cleanly in
Excel, Numbers and Google Sheets and be immediately sortable.

## Experience

### 1. Pick a Pokémon
A single search box, front and centre, on an otherwise near-empty page. Typing shows autocomplete
suggestions with a sprite and National Dex number; matching forgives casing, accents and partial
names ("char" → Charizard, Charmander, Charmeleon).

### 2. Preview the list
Before downloading anything, the user sees the full card list on screen. This is what turns a blind
download into a confident one — it answers "is this actually complete?" and "is this what I expected?"
before someone commits 24 pages of paper and an hour of cutting.

```
Charizard · English · with variants        214 cards · ~24 pages
──────────────────────────────────────────────────────────────────
 [img]  Base Set          4/102   Holo Rare      Unlimited   1999
 [img]  Base Set          4/102   Holo Rare      Shadowless  1999
 [img]  Base Set          4/102   Holo Rare      1st Edition 1999
 [img]  Base Set 2        4/130   Holo Rare      Unlimited   2000
 [img]  Team Rocket       4/82    Holo Rare      Dark Chariz 2000
 [img]  Gym Challenge     3/132   Holo Rare      Blaine's    2000
                                                    … 208 more
──────────────────────────────────────────────────────────────────
   [ Printable sheets ]   [ PDF overview ]   [ CSV ]
```

- A **compact list with thumbnails** — one row per card variant, showing the same fields the PDF and
  CSV will contain. It's a preview of the data, so it should look like the data.
- Ordered chronologically by set release, matching the print output. What you see is the order you
  get.
- The generation options (language, variants on/off) sit above the list and **update it live**, so
  the effect of toggling variants is immediately visible in both the list and the page count.
- Long lists are paginated or lazily loaded rather than dumping 214 rows at once, but the full list
  must be reachable — a preview that hides most of the list can't answer the completeness question.

Deliberately *not* here: sorting, filtering, per-card selection, or ticking cards off. Those turn a
confirmation step into a workspace and pull the product back toward the digital-tracking model this
version deliberately walked away from.

### 3. Pick a format
Three clear download options below the list, each with a one-line description of what it's for, and
the card and page count alongside ("214 cards · ~24 pages").

Page count matters here — printing 24 pages of placeholders is a real decision, and surprising
someone with it is a bad experience.

### 4. Download
The file downloads. Nothing else happens. No signup prompt, no upsell, no "create an account to
save this".

## Options at generation time

Kept deliberately minimal — every option is a decision the user has to make before they get their
file.

- **Language** — English or Japanese (see open questions on combining them)
- **Include variants** — whether separate printings (1st Edition, Shadowless, Reverse Holo…) each
  get their own placeholder, or one placeholder per card. This is the option most likely to change
  the page count dramatically, so its effect on the count must be visible immediately.

Nothing else. No rarity filters, no era pickers — those belong to a browsing product, and this isn't
one.

## Requirements

### Must have
1. Find any Pokémon by name via typeahead search.
2. The card list is **complete** — every card of that Pokémon in the chosen language, with every
   known printing variant. Completeness is the product's entire value.
3. Preview the full card list on screen before downloading, with thumbnails, in the same order and
   with the same fields as the downloads.
4. Download printable sheets with card images at true 63 × 88 mm size, with cut lines.
5. Download a PDF overview table of the same list.
6. Download a CSV of the same list, cleanly openable in common spreadsheet tools.
7. Card count and estimated page count shown alongside the preview, before download.
8. Print instructions that defend against printer scaling, with a way to verify true size.
9. Whole flow completable with no account and no more than a search plus a click.

### Should have
10. Language option (EN / JP).
11. Variants on/off option, updating the preview list and page count live.
12. Shareable URL per Pokémon so a previewed list can be linked or bookmarked.

### Won't have (v1)
13. Sorting, filtering or per-card selection in the preview; any digital ownership tracking, accounts, prices, gameplay data, cameo/appearance cards,
    languages beyond EN/JP, custom binder layout pickers.

## Success Criteria

- A collector goes from landing page to downloaded file in under a minute, preview included.
- The preview is trusted: users who reach it go on to download, rather than bouncing because the
  list looked wrong or incomplete.
- Printed placeholders fit standard binder pockets without trimming or reprinting — measured by
  actually printing and slotting them, on more than one printer.
- Zero reported missing printings for the most-collected Pokémon.
- The CSV opens correctly in Excel, Numbers and Google Sheets with no manual fixing.

## Open Questions

1. **Placeholder art vs. real art.** Should placeholders show the real card image, or a greyed /
   outlined version that reads clearly as "not owned yet" when flipping the binder? The greyed
   version is arguably the better artifact and also sidesteps any question of printing full-quality
   card reproductions.
2. **Print scale verification.** What's the mechanism — a printed ruler strip, a "measure this box"
   mark, both? Needs a decision, because requirement #7 hinges on it.
3. **Variant taxonomy.** What counts as a distinct printing worth its own placeholder (error prints?
   stamp variations?) directly sets the page count. Needs a stated, defensible definition.
4. **Mixing languages.** Can a collector generate one sheet containing both EN and JP cards, or is
   it strictly one language per download?
5. **Multi-Pokémon cards.** Do cards featuring two Pokémon (e.g. Pikachu & Zekrom GX) appear in both
   Pokémon's sheets? Presumed yes.
6. **Image rights.** Printing card art at true size is close to reproducing the card. Worth a
   deliberate position before launch — open question 1 may resolve this too.
7. **Data source.** Not a product question, but completeness and variant granularity are
   product-defining, and available sources will constrain requirement #2. Worth validating first.

---

*Next step: resolve open questions 1 and 2 — they define what the core output physically looks like.
Then data source validation, which is the critical path.*
