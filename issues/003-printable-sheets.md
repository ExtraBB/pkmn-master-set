# 003 — Printable placeholder sheets

**Priority:** P0 · **Depends on:** 001 · **Validated by:** 005

## The need

> As a binder collector, I want to print and cut out placeholder cards for everything I don't own
> yet, so my binder shows the complete set and I can see the gaps at a glance while flipping
> through it.

This is the core output. The other two downloads are conveniences; this is the reason the product
exists.

## What this covers

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

- **True trading-card size — 63 × 88 mm.** A cut-out must sit correctly in a standard binder pocket.
  This is the quality bar for the whole issue.
- **Image-led.** The art is the identifier; a collector should recognise the missing card while
  flipping, without reading.
- **Minimal caption** on each placeholder: set, card number, variant.
- **Cut lines** between cards, with as many cards per page as fit at true size.
- **Chronological by set release**, so the binder tells the Pokémon's history front to back.

## Two decisions inside this issue

**Placeholder art treatment.** Real card image, or a greyed/outlined version that reads clearly as
"not owned yet" when flipping the binder? The greyed version is arguably the better artifact — a gap
should look like a gap — and it also sidesteps reproducing full-quality card art at exact card size,
which is worth a deliberate position before launch.

**Scale defence.** Browser and printer "fit to page" scaling silently resizes the page, and the
cut-outs come out wrong. The user discovers this only after cutting a whole stack. The sheet needs
both clear print instructions and something printed on the page — a ruler strip, a measure-this-box
mark — so scale can be verified before any cutting starts. The exact mechanism is open.

## Acceptance criteria

- Printed cards measure 63 × 88 mm on paper, verified with a ruler, on more than one printer
- The page carries a scale-verification mark the user can check before cutting
- Print instructions warn about fit-to-page scaling in plain language, where the user will see them
- Cut lines produce clean, correctly-sized cards without trimming twice
- A placeholder is recognisable at a glance in a binder pocket
- Both A4 and Letter produce correctly-sized cards
