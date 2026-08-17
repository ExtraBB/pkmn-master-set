# 002 — Search a Pokémon and preview its card list

**Priority:** P0 · **Depends on:** 001

## The need

> As a collector, I want to type a Pokémon's name and immediately see the full list I'd be
> downloading, so I can confirm it's complete and correct before committing paper and an hour of
> cutting.

This is the entire web product. Everything else is a file.

## What this covers

**The landing page.** Near-empty, a single prominent search box. No marketing, no feature tour.

**Typeahead search.** Suggestions with a sprite and National Dex number, forgiving of casing,
accents and partial names — "char" finds Charizard, Charmander and Charmeleon.

**The preview list.** A compact list with thumbnails, one row per card variant, showing the same
fields in the same chronological order as the downloads. What you see is what you get.

```
Charizard · English · with variants        214 cards · ~24 pages
──────────────────────────────────────────────────────────────────
 [img]  Base Set          4/102   Holo Rare      Unlimited   1999
 [img]  Base Set          4/102   Holo Rare      Shadowless  1999
 [img]  Base Set          4/102   Holo Rare      1st Edition 1999
 [img]  Base Set 2        4/130   Holo Rare      Unlimited   2000
                                                    … 210 more
──────────────────────────────────────────────────────────────────
   [ Printable sheets ]   [ CSV ]
```

**Counts before commitment.** Card count and estimated page count, visible next to the download
buttons. Printing 24 pages is a real decision and surprising someone with it is a bad experience.

**Generation options.** Language (EN/JP) and variants on/off, sitting above the list, updating both
the list and the page count live. The variants toggle can double the page count — that effect should
be visible, not discovered after printing.

**Shareable URL** per previewed list, carrying the chosen options.

**Empty and error states.** Unknown Pokémon, a Pokémon with no cards, and data temporarily
unavailable each need an honest message rather than an empty page.

## Acceptance criteria

- Landing page to a previewed list in one search and one click
- The full list is reachable — long lists may paginate or load lazily, but a preview that hides most
  of the list can't answer the completeness question
- Preview order and fields match the downloads exactly
- Toggling variants visibly changes both the list and the page count
- No account, no signup prompt, no upsell anywhere in the flow

## Explicitly not in this issue

No sorting, no filtering, no per-card selection, no ticking cards off. The preview is a confirmation
step, not a workspace — those features turn it into the digital-tracking product this version
deliberately walked away from.
