// Command sprites downloads the National Dex sprites the typeahead shows next to
// each suggestion.
//
// Like cmd/species this is a build-time tool, run by hand and its output
// committed: PokéAPI is a generation-time dependency only, the server never calls
// it, and a suggestion list must not depend on a third party being up.
//
//	go run ./cmd/sprites -out web/static/sprites
//
// Rerun it after regenerating the species index.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// spriteURL is PokéAPI's 96x96 front-default sprite: ~0.5-1.7KB each, and the one
// set with complete coverage of the whole Dex.
const spriteURL = "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/%d.png"

// workers bounds concurrency: this hits a public CDN a thousand times and there is
// no reason to be rude about it.
const workers = 8

func main() {
	out := flag.String("out", "web/static/sprites", "output directory")
	species := flag.String("species", "internal/cards/data/species.json", "species index to read dex numbers from")
	force := flag.Bool("force", false, "redownload sprites that already exist")
	flag.Parse()

	// The dex numbers come from the committed species index rather than a second
	// PokéAPI call, so the sprite set and the searchable set cannot drift apart.
	ids, err := dexIDs(*species)
	if err != nil {
		log.Fatalf("read species index: %v", err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatalf("create %s: %v", *out, err)
	}

	fetched, skipped, missing := download(ids, *out, *force)

	// A partial sprite set would ship as silently broken suggestion rows, so it
	// fails the regeneration instead.
	if have := len(ids) - len(missing); have < 900 {
		log.Fatalf("only %d of %d sprites present, expected the full National Dex", have, len(ids))
	}
	fmt.Printf("%d downloaded, %d already present, %d unavailable\n", fetched, skipped, len(missing))
	for _, id := range missing {
		fmt.Printf("  no sprite for #%d\n", id)
	}
}

func dexIDs(path string) ([]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var list []struct {
		DexID int `json:"dexId"`
	}
	if err := json.NewDecoder(f).Decode(&list); err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(list))
	for _, s := range list {
		if s.DexID > 0 {
			ids = append(ids, s.DexID)
		}
	}
	if len(ids) < 900 {
		return nil, fmt.Errorf("species index holds only %d entries", len(ids))
	}
	return ids, nil
}

func download(ids []int, dir string, force bool) (fetched, skipped int, missing []int) {
	client := &http.Client{Timeout: 30 * time.Second}

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		next = make(chan int)
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range next {
				path := filepath.Join(dir, strconv.Itoa(id)+".png")
				if !force {
					if _, err := os.Stat(path); err == nil {
						mu.Lock()
						skipped++
						mu.Unlock()
						continue
					}
				}
				err := fetch(client, id, path)
				mu.Lock()
				switch {
				case err == nil:
					fetched++
				default:
					// A Pokémon with no sprite upstream is a gap, not a failure:
					// the suggestion row still renders, just without art.
					log.Printf("#%d: %v", id, err)
					missing = append(missing, id)
				}
				mu.Unlock()
			}
		}()
	}
	for _, id := range ids {
		next <- id
	}
	close(next)
	wg.Wait()
	return fetched, skipped, missing
}

// fetch writes the sprite to a temporary file first, so an interrupted run never
// leaves a truncated PNG behind for a later run to skip over as "already present".
func fetch(client *http.Client, id int, path string) error {
	resp, err := client.Get(fmt.Sprintf(spriteURL, id))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pokeapi sprites: %s", resp.Status)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".sprite-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
