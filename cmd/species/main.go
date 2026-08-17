// Command species regenerates the National Dex index used by the typeahead.
//
// This is a build-time tool, run by hand and its output committed. PokéAPI is a
// generation-time dependency only: the server never calls it, and it never
// becomes a Go module.
//
//	go run ./cmd/species -out internal/cards/data/species.json
//
// Rerun it when a new generation of Pokémon is released.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const pokeAPISpecies = "https://pokeapi.co/api/v2/pokemon-species?limit=100000"

type speciesOut struct {
	DexID  int    `json:"dexId"`
	Name   string `json:"name"`
	NameJA string `json:"nameJa,omitempty"`
}

func main() {
	out := flag.String("out", "internal/cards/data/species.json", "output path")
	flag.Parse()

	list, err := fetchSpecies()
	if err != nil {
		log.Fatalf("fetch species: %v", err)
	}
	if len(list) < 900 {
		// A truncated list would silently make Pokémon unsearchable, which is
		// worse than failing the regeneration.
		log.Fatalf("got only %d species, expected the full National Dex", len(list))
	}

	if err := write(*out, list); err != nil {
		log.Fatalf("write: %v", err)
	}
	fmt.Printf("wrote %d species to %s\n", len(list), *out)
}

func fetchSpecies() ([]speciesOut, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(pokeAPISpecies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pokeapi: %s", resp.Status)
	}

	var body struct {
		Results []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	list := make([]speciesOut, 0, len(body.Results))
	for _, r := range body.Results {
		id := dexIDFromURL(r.URL)
		// Alternate forms live above 10000 and are not separate Dex entries.
		if id == 0 || id > 10000 {
			continue
		}
		list = append(list, speciesOut{DexID: id, Name: displayName(r.Name)})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].DexID < list[j].DexID })
	return list, nil
}

// dexIDFromURL pulls the trailing id out of ".../pokemon-species/6/".
func dexIDFromURL(u string) int {
	parts := strings.Split(strings.Trim(u, "/"), "/")
	if len(parts) == 0 {
		return 0
	}
	id, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return id
}

// displayName restores the capitalisation and punctuation PokéAPI strips.
//
// PokéAPI serves lowercase hyphenated slugs, which is not what a collector types
// or reads. Most are two words ("great-tusk" -> "Great Tusk"); the ones whose real
// name keeps a hyphen or uses punctuation the slug threw away are listed
// explicitly, because each is a name someone will actually search for.
func displayName(slug string) string {
	if n, ok := specialNames[slug]; ok {
		return n
	}
	words := strings.Split(slug, "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// specialNames covers every Pokémon whose real name cannot be recovered from its
// slug by title-casing.
var specialNames = map[string]string{
	"nidoran-f": "Nidoran♀",
	"nidoran-m": "Nidoran♂",
	"mr-mime":   "Mr. Mime",
	"mr-rime":   "Mr. Rime",
	"mime-jr":   "Mime Jr.",
	"farfetchd": "Farfetch'd",
	"sirfetchd": "Sirfetch'd",
	"ho-oh":     "Ho-Oh",
	"porygon-z": "Porygon-Z",
	"porygon2":  "Porygon2",
	"jangmo-o":  "Jangmo-o",
	"hakamo-o":  "Hakamo-o",
	"kommo-o":   "Kommo-o",
	"type-null": "Type: Null",
	"flabebe":   "Flabébé",
	"chi-yu":    "Chi-Yu",
	"chien-pao": "Chien-Pao",
	"ting-lu":   "Ting-Lu",
	"wo-chien":  "Wo-Chien",
}

func write(path string, list []speciesOut) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", " ")
	enc.SetEscapeHTML(false)
	return enc.Encode(list)
}
