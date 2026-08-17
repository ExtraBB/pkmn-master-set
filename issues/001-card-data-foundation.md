# 001 — Get the card data right

**Priority:** P0 · **Blocks:** 002, 003, 004

## The need

> As a collector, I need the list to contain *every* card of my Pokémon, so I can trust that what I
> print is the real target and not a partial list that will embarrass me later.

Completeness is the product's entire value. Every other issue is a rendering of this data. A missing
printing isn't a small bug — it means a collector cuts out and files a set of placeholders that is
quietly wrong, and they only find out months later.

## What this covers

**The completeness bar.** A written definition of what "complete" means, and verification that we
actually meet it for the most-collected Pokémon. Spot-checking a handful of Pokémon against what
collectors themselves consider the full list.

**The variant taxonomy.** The unit of this product is the printing, not the card — *Base Set
Charizard 1st Edition* is a different thing to print than *Base Set Charizard Unlimited*. We need a
stated position on each category:

- Print runs — 1st Edition, Shadowless, Unlimited
- Finishes — non-holo, holo, reverse holo, pattern holos
- Stamps — Staff, Prerelease, League, event stamps
- Error prints — miscuts, missing symbols, corrected reprints
- Promo re-releases of the same art

**Per-card fields.** Every output shows the same fields, so they're defined once here: card name,
set, card number and set total, rarity, variant/finish, language, release date, illustrator.

**Two attribution rules:**
- Do cards featuring two Pokémon (e.g. Pikachu & Zekrom GX) appear in both Pokémon's lists?
  Presumed yes.
- Can one list mix English and Japanese, or is it strictly one language per list?

## Acceptance criteria

- A written taxonomy answers each variant category above with a yes/no and a one-line rationale
- The taxonomy is visible to users, so a collector who disagrees can see what we counted and why
- Categories we deliberately exclude are named explicitly rather than silently omitted — a collector
  should never wonder whether we forgot something or excluded it on purpose
- For at least five heavily-collected Pokémon, the list matches a collector-sourced reference with
  no missing printings
- Every field above is present for every card, or its absence is visible rather than blank

## Why it's first

The variant taxonomy sets the page count, which sets what the printable sheet even looks like. If
variant-level data turns out to be unavailable at the granularity this PRD assumes, issues 003 and
004 change shape. Better to learn that now than after the print layout is built.
