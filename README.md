# pkmn-master-set

Go + htmx skeleton. The server renders full pages and HTML fragments with
`html/template`; htmx swaps the fragments into the page.

## Run

```sh
go run .          # http://localhost:8080
ADDR=:3000 go run .
```

## Test

```sh
go test ./...
```

## Layout

```
main.go                     entrypoint, graceful shutdown
internal/server/server.go   routes + handlers
web/embed.go                embeds templates and static assets into the binary
web/templates/              index.html (page), ping.html (htmx fragment)
web/static/                 app.css, vendored htmx.min.js
```

## Adding a feature

1. Add a handler in `internal/server/server.go` and register it in `Routes()`.
2. Add a fragment template in `web/templates/` wrapped in
   `{{define "name.html"}}...{{end}}`.
3. Point an element at it with `hx-get`/`hx-post` plus `hx-target`.

htmx is vendored at `web/static/js/htmx.min.js` (v2.0.4) — no CDN at runtime.
Templates are embedded, so restart the server to pick up template edits.
