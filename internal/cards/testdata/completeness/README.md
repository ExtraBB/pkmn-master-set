# Completeness fixtures

These files are the collector-sourced reference the card list is checked against. They exist because
completeness is the product's entire value: a missing printing means someone cuts out and files a set
of placeholders with a hole in it, and only finds out months later.

One file per heavily-collected Pokémon. Each line is:

```
<tcgdex-set-id> <card-number>      # optional comment
```

Blank lines and `#` comments are ignored. Card numbers are compared with leading zeros stripped, so
`021` and `21` are the same card.

`TestCompletenessAgainstCollectorReference` (build tag `live`) asserts every entry here is **present**
in our generated list:

```sh
go test -tags=live -run TestCompleteness ./internal/cards/
```

Pokémon TCG Pocket sets (TCGdex serie `tcgp`: `A1`, `A2b`, `B1a`, `P-A`, …) must **not** appear in
these files. Those cards are digital-only and the catalog excludes the whole serie, so a `tcgp` line
here is a guaranteed failure rather than a real gap.

It is a **subset** check. Extra cards in our list are fine — these references are transcribed by hand
and lag new sets. A card the reference has and we don't is the failure worth catching.

## How these were built, and what the test does and doesn't prove

Each file is the generated list as it stood on 2026-08-17, **after** it was reconciled line by line
against the Pokémon's Bulbapedia `(TCG)` page. Everything Bulbapedia lists in English is in the file;
everything Bulbapedia lists that our source cannot supply is named in the file's header under
*KNOWN UPSTREAM GAPS*, rather than quietly left out.

So the test is a **regression guard**, not an independent oracle: it proves the list has not lost
anything since it was last reconciled by hand. The reconciliation itself is the human step, and it
has to be redone when the header's date gets old.

## Refreshing a fixture

1. Print the current list: `go test -tags=live -run TestLiveChecklist -v ./internal/cards/`
2. Compare against the Pokémon's page on Bulbapedia (the title is `<Pokémon>_(TCG)` — note that
   `<Pokémon>_(Pokémon)/TCG` 404s). Fetching `?action=raw` and parsing the `{{card list/release}}`
   rows with an `enset=` field is far more reliable than reading the rendered page.
3. Replace the card lines, update the header date, and move anything newly available out of
   *KNOWN UPSTREAM GAPS*.

Only add a line you have actually verified. A fabricated entry turns this test into noise, which is
worse than not having it.
