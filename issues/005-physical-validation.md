# 005 — Prove it works on a real printer and a real binder

**Priority:** P0 · **Depends on:** 003

## The need

> As a collector, I need the placeholders I print to actually fit my binder pockets, because I find
> out they don't only after cutting out sixty of them.

This product's output leaves the screen and becomes a physical object. Nothing on screen can tell us
whether it worked. This issue is the loop that closes that gap.

## Why it's a separate issue

Every other issue can be verified by looking at it. This one can only be verified by printing,
cutting, and slotting cards into a real binder. It's the difference between a product that
technically generates a PDF and one that a collector can actually use — and it's the failure mode
most likely to slip through review, because it looks fine on screen right up until it doesn't fit.

## What this covers

**Print across real conditions:**
- More than one printer, including at least one cheap home inkjet
- A4 and Letter paper
- Printing from a downloaded file and directly from a browser
- Default printer settings, not carefully-tuned ones — this is how users will actually print

**Then physically check:**
- Cards measure 63 × 88 mm with a ruler
- Cut-outs sit correctly in a standard 9-pocket binder page without trimming or forcing
- The scale-verification mark on the page does its job — someone following the instructions catches
  a mis-scaled print *before* cutting
- A placeholder is recognisable at a glance when flipping through the binder

**And check the failure path:** deliberately print with fit-to-page enabled and confirm the user
would notice before cutting. If they wouldn't, the scale defence in 003 isn't finished.

## Acceptance criteria

- A test print has been cut and slotted into a real binder page, by hand, at least once
- Correct sizing confirmed on at least two different printers with default settings
- A deliberately mis-scaled print is caught by the verification mark
- Findings feed back into 003 rather than being logged and forgotten

## Note

Start this early. The first test print will teach us more than any amount of specification, and it's
cheap — one page, one pair of scissors.
