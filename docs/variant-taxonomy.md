# Variant taxonomy

*What counts as a card, and what counts as a separate printing.*

This document is the stated position behind every list this product generates. It exists so a
collector who disagrees with a count can see exactly what we counted and why, rather than guessing
whether something is missing or deliberately left out.

## The unit is the printing, not the card

*Base Set Charizard 1st Edition* and *Base Set Charizard Unlimited* are two different things to hunt
down and two different slots in a binder. So the row — and the placeholder you print — is the
**printing**, not the card.

The rule is deliberately simple and inclusive:

> **Every distinct printing our data source records for a card gets its own row.**

Turning **Include variants** off collapses each card back to a single row. Nothing is deleted when
you do that: the row reports how many printings it stands for, so the count stays honest in both
directions.

## What counts

| Category | Counted? | Why |
| --- | --- | --- |
| **Print runs** — 1st Edition, Shadowless, Unlimited, 1999–2000 Copyright, No Rarity Symbol, Blue Border | **Yes** | These are the distinctions collectors organise around and pay different prices for. A binder needs a slot for each. |
| **Finishes** — non-holo (*Standard*), *Holo*, *Reverse Holo*, pattern holos, *Lenticular* | **Yes** | A Reverse Holo is a separate card to find, not a cosmetic detail. This is also the single biggest driver of page count. |
| **Stamps** — 1st Edition, Staff, Prerelease, League, Championship, event and promo stamps | **Yes**, where the source records a stamp | A stamped card is a distinct object with its own print run. ~35 distinct stamps appear in the data. |
| **Error prints** — corrected reprints and symbol errors that shipped as their own print run (e.g. the Energy Symbol Error) | **Yes**, where the source models them as a variant | These were printed as a distinct run, so they are findable and countable. |
| **Promo re-releases of the same art** | **Yes** | A different set and a different card number means a different card to hunt, even when the illustration is identical. |
| **Cards featuring two Pokémon** (e.g. Pikachu & Zekrom GX) | **Yes — in both lists** | The card genuinely belongs to both collections. Our source tags such cards with both National Dex numbers, so Pikachu & Zekrom GX appears when you search Pikachu *and* when you search Zekrom. |
| **Jumbo / oversized cards** | **Yes**, labelled *Jumbo* | They exist and collectors track them. They will not fit a standard 63 × 88 mm pocket, so the label warns you before you cut. |

## What we deliberately exclude

Named explicitly, so you never have to wonder whether we forgot.

| Excluded | Why |
| --- | --- |
| **Miscuts, off-centre prints, ink defects and other physical accidents** | Not a print run — a one-off accident. There is no enumerable list of them, so no honest placeholder can be generated. |
| **Grading, condition and provenance** | A PSA 10 and a played copy are the same printing. Condition is about the individual object, not about what exists. |
| **Trainer and Energy cards that depict a Pokémon** | The card is not *of* that Pokémon. Our source does not tag them with a Dex number, and the PRD puts cameo and appearance cards out of scope. |
| **Sealed product, tins, jumbo boxes, coins and inserts** | Not cards. |
| **Prices** | The product is about what exists, not what it costs. |
| **Gameplay data** — HP, attacks, abilities | Not something a placeholder or a checklist needs. |
| **Languages other than English and Japanese** | Out of scope for v1. |

## Fields on every row

Every output — the on-screen preview, the printable sheets, the PDF and the CSV — carries the same
fields, in the same order:

| Field | Notes |
| --- | --- |
| Card name | As printed, so *Blaine's Charizard* and *Dark Charizard* read as themselves |
| Set | Set name |
| Card number and set total | e.g. `4/102` |
| Rarity | e.g. Holo Rare |
| Variant / finish | The printing label, e.g. *1st Edition · Shadowless · Holo* |
| Language | English or Japanese |
| Release date | Set release date |
| Illustrator | |

**When a field is unknown, it says "unknown".** It never renders as an empty cell. A blank reads as a
bug; a stated "unknown" reads as what it is — a gap in the source data.

## Ordering

Every list is ordered **chronologically by set release date**, then by card number, then by printing
(earliest print run first). A filled binder therefore reads as the Pokémon's history from front to
back, and what you see in the preview is the order you get on paper.

Sets whose release date is unknown (a handful of promos) sort to the **end** rather than the start,
so one dateless promo cannot corrupt the chronology.

## Known limitations of the source

These are limits of the available data, not decisions:

- **Japanese coverage is materially thinner than English.** For Charizard our source lists 124 English
  cards and 25 Japanese ones. The Japanese list is real but incomplete, and should not yet be treated
  as a full master set.
- **Stamps and error prints are recorded unevenly.** Where the source does not record a stamp
  (many Staff, Prerelease and League stampings), we cannot list it. This is a gap in coverage, not a
  decision to exclude the category — the category is counted, per the table above.
- **A card with no National Dex number attached cannot be attributed to a Pokémon.** Cards are placed
  in your list by Dex number, so a card the source has not tagged is invisible to us. As of
  2026-08-17 this affects 66 of 19,870 Pokémon cards (0.33%) — the whole of the *Ascended Heroes*
  set, which is newly released and not yet tagged. A test watches this figure and fails if it grows
  past 1%.
- **Some sets are absent from the source entirely**, so no card from them can appear. Currently that
  includes box toppers, the Pokémon TCG Classic decks, the 30th Celebration set, and parts of the
  Trainer Kit and MEP promo sets.
- **A printing our source has never seen cannot appear in your list.** If you know of a missing
  printing, that is a data bug worth reporting.

### How we check

Lists for Charizard, Pikachu, Eevee, Mewtwo and Umbreon are reconciled by hand against Bulbapedia and
kept as fixtures, so the list cannot quietly lose printings between releases. See
`internal/cards/testdata/completeness/`. Every English card Bulbapedia lists for those five is in our
list, apart from the source gaps named above — which are written down rather than dropped.

## One language per list

A generated list is **English or Japanese, never both**. Mixing them would make the card number and
set columns ambiguous and would double the page count without saying so. Generate one of each if you
collect both.

## Source

Card data comes from [TCGdex](https://tcgdex.dev). Lists are fetched live and cached, so a newly
released set appears without a redeploy.
