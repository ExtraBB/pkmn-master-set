package cards

import (
	"bytes"
	_ "embed"
	"fmt"
)

// speciesJSON is the generated National Dex index. It is embedded rather than
// fetched so search works with no network and no startup latency; regenerate with
// `go run ./cmd/species`.
//
//go:embed data/species.json
var speciesJSON []byte

// EmbeddedSpecies returns the built-in species index.
func EmbeddedSpecies() (*SpeciesIndex, error) {
	idx, err := LoadSpecies(bytes.NewReader(speciesJSON))
	if err != nil {
		return nil, fmt.Errorf("cards: loading embedded species index: %w", err)
	}
	return idx, nil
}
