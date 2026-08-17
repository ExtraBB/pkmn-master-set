# Issues

Product issues derived from [`PRD.md`](../PRD.md). Product perspective only — no technical design.

Five slices, in dependency order. Each states the user need, why it matters, and how we'll know it's
done. Implementation approach is deliberately left open.

| # | Issue | Priority |
| --- | --- | --- |
| [001](001-card-data-foundation.md) | Get the card data right | P0 |
| [002](002-search-and-preview.md) | Search a Pokémon and preview its card list | P0 |
| [003](003-printable-sheets.md) | Printable placeholder sheets | P0 |
| [004](004-csv-overview.md) | CSV overview | P1 |
| [005](005-physical-validation.md) | Prove it works on a real printer and a real binder | P0 |

## Sequencing

001 comes first and genuinely blocks the rest — every output is a rendering of that data, and its
variant granularity decides what the other four issues are even shaping. 005 is last to *complete*
but should start early: the first test print is worth more than any amount of specification.

## Out of scope for v1

Digital ownership tracking, accounts, price history/portfolio value, gameplay data, cameo cards,
languages beyond EN/JP, sorting/filtering in the preview. See the PRD's "Explicitly not this product"
table for reasoning.
