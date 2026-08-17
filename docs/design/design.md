# pkmn-master-set — design

Screens for the v2 PRD: search → preview → download. Static screens, light and dark.
Sample data is Charizard, as in the PRD.

## Files

| File | Contents |
| --- | --- |
| `screenshots/light-01-landing.png` | Landing — single search box |
| `screenshots/light-02-preview-variants-on.png` | Preview, variants on (214 cards · ~24 pages) |
| `screenshots/light-03-variants-off.png` | Preview, variants off (71 cards · ~8 pages) |
| `screenshots/dark-*.png` | Same three screens, dark mode |

## Palette

Light and dark are the same system, not two designs. Neutrals carry the layout; one accent
carries action.

**Light**

- Page `#F8FAFF`, surface `#FEFDFF`, desk `#E8EFF9`
- Borders `#D4DFF0` (default), `#E8EFF9` (subtle)
- Text `#001432` primary, `#32486B` secondary, `#4F6486` tertiary, `#6E819E` muted
- Action `#191FBB`, with `#A2BCFF` / `#D0DEFF` / `#F3F7FF` for outlines and tints
- Accent `#00D9C6`, used for the primary download and the count-delta pill

**Dark**

- Page `#051C43`, surface `#092045`, desk `#001432`
- Borders `#32486B` (default), `#182E52` (subtle)
- Text `#FEFDFF` primary, `#D4DFF0` secondary, `#AFBFD8` tertiary, `#8DA0BB` muted
- Teal is the only colour: `#00D9C6` for primary action and toggles, `#84F5E5` for secondary
  action text, `#01544C` for outlines, `#002824` for tinted pills. Indigo is not used.

## Type

- Display / headings: **Jost** Medium 500, tracking `-0.01em`. Hero 56px, screen title 40px,
  section titles 20–26px.
- UI and body: **Inter** 400/500. Body 16px, table rows 14–15px, labels 11–13px.
- Table column headers: Inter 500 / 11px, uppercase, `0.06em` tracking.

## Screen notes

**01 Landing.** One search box on a near-empty page, per the PRD's flow requirement. Below it,
four example Pokémon and a single footer line naming the three outputs. Nothing else competes
with the search field.

**02 Preview, variants on.** Card count and page estimate sit in the header, at the same
weight as the Pokémon name, because 24 pages of paper is the real decision on this screen.
Options (language, include variants) sit in one tinted bar directly above the list. The table
shows the exact fields the PDF and CSV will contain — set, number, rarity, variant, release
year — in chronological set order, matching the print output. The footer says "Showing 8 of
214 — every printing is in the file" so the truncation doesn't read as an incomplete list.

Two affordances in the list are read-only, and deliberately quiet. The set name links to the
card on Bulbapedia in a new tab — collectors verify there, so the preview should not be a dead
end — styled as the cell's own text with a hairline underline, because a whole column of
action-coloured links would compete with the download. And the thumbnail opens the card at
readable size, because 44 px cannot answer "is that the printing I think it is". Neither adds
state: nothing is sorted, filtered, selected or ticked off.

The three download options are weighted, not equal: printable sheets is the core output and
gets the filled dark card; PDF and CSV are outlined. Each carries its own count ("214 cards ·
24 pages", "2 pages", "214 rows").

Under them, the print-scale defence: a fixed instruction to print at 100% plus a 50 mm
reference strip. Open question 2 in the PRD is unresolved — this draft assumes a printed
ruler strip rather than a "measure this box" mark. The strip can be toggled off in the design.

**03 Variants off.** Same screen, one placeholder per card. The variant column drops out, the
count falls to 71 cards / 8 pages, and a teal pill states the delta (−143 cards · −16 pages)
so the effect of the toggle is legible immediately, per requirement 11.

## Deliberately absent

No sort, no filter, no per-card selection, no checkboxes, no account prompt — the preview is a
confirmation step, not a workspace. (Cardmarket/TCGplayer price columns were added after this
design pass and are not shown in the screenshots above; they follow the same read-only,
quietly-styled link treatment as the set-name link described in screen 02.)

## Placeholders

Card thumbnails are striped placeholders. Real card art (or greyed art, pending open question
1) replaces them. Fonts: Jost and Inter variable files in `fonts/`.
