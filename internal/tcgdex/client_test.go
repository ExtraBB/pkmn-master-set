package tcgdex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// noBackoff keeps retry tests instant.
func noBackoff(int) time.Duration { return 0 }

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(WithBaseURL(srv.URL), WithBackoff(noBackoff))
}

// The detail shape the real API returns. tcgdex prices variants_detailed
// entries individually rather than the card as a whole: real Base Set Charizard
// data has a tracked listing for the Unlimited printing but none for Shadowless
// or 1st Edition, so this fixture mirrors that — one priced entry, two with a
// null "pricing".
const charizardDetail = `{
  "id":"base1-4","localId":"4","name":"Charizard","category":"Pokemon","dexId":[6],
  "rarity":"Rare","illustrator":"Mitsuhiro Arita",
  "image":"https://assets.tcgdex.net/en/base/base1/4",
  "variants":{"normal":false,"reverse":false,"holo":true,"firstEdition":true,"wPromo":false},
  "variants_detailed":[
    {"type":"holo","subtype":"unlimited","size":"standard","variantId":"4ffrmhcf",
     "pricing":{"cardmarket":{"unit":"EUR","idProduct":483559,"avg":249.99,"avg-holo":312.4},
       "tcgplayer":{"unit":"USD","updated":"2026-08-17T00:00:00Z",
         "holofoil":{"productId":219333,"marketPrice":312.4}}}},
    {"type":"holo","subtype":"shadowless","stamp":["1st-edition"],"size":"standard","variantId":"mtltux8q",
     "pricing":{"cardmarket":null,"tcgplayer":null}},
    {"type":"holo","subtype":"shadowless","size":"standard","variantId":"3taksсxp"}],
  "set":{"id":"base1","name":"Base Set","cardCount":{"official":102,"total":102}}
}`

// restServer answers the two endpoints a card lookup uses: the brief list and one
// detail per id. It records every path it was asked for, so a test can assert the
// client never reaches for another transport.
func restServer(t *testing.T, list string, detail map[string]string) (*Client, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var paths []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.String())
		mu.Unlock()

		id := strings.TrimPrefix(r.URL.Path, "/en/cards/")
		if body, ok := detail[id]; ok {
			w.Write([]byte(body))
			return
		}
		w.Write([]byte(list))
	})
	return c, &paths
}

func TestCardsByDexUsesREST(t *testing.T) {
	c, paths := restServer(t, `[{"id":"base1-4"}]`, map[string]string{"base1-4": charizardDetail})

	got, err := c.CardsByDex(context.Background(), "en", 6)
	if err != nil {
		t.Fatalf("CardsByDex: %v", err)
	}

	// The strict eq: filter matters: the API's default laxist match compares as a
	// substring, so a bare dexId=6 would also return dex 16, 60 and 106.
	want := []string{"/en/cards?dexId=eq:6", "/en/cards/base1-4"}
	if len(*paths) != len(want) {
		t.Fatalf("requested %v, want %v", *paths, want)
	}
	for i, p := range *paths {
		if p != want[i] {
			t.Errorf("request %d = %q, want %q", i, p, want[i])
		}
	}

	if len(got) != 1 {
		t.Fatalf("got %d cards, want 1", len(got))
	}
	card := got[0]
	if card.ID != "base1-4" || card.Number != "4" || card.SetTotal != 102 {
		t.Errorf("card = %+v", card)
	}
	if len(card.DexIDs) != 1 || card.DexIDs[0] != 6 {
		t.Errorf("DexIDs = %v, want [6]", card.DexIDs)
	}
	if len(card.Variants) != 3 {
		t.Fatalf("got %d variants, want 3", len(card.Variants))
	}
	// subtype and stamp are what separate Base Set Charizard's three printings, so
	// they have to survive the REST decode intact.
	if card.Variants[1].Subtype != "shadowless" || len(card.Variants[1].Stamp) != 1 {
		t.Errorf("variant 1 = %+v, want shadowless with a stamp", card.Variants[1])
	}
	// The Unlimited printing has a tracked listing...
	unlimited := card.Variants[0].Pricing
	if unlimited.Cardmarket.IDProduct != 483559 || unlimited.Cardmarket.Avg != 249.99 {
		t.Errorf("unlimited cardmarket pricing = %+v", unlimited.Cardmarket)
	}
	if f := unlimited.TCGPlayer.Finishes["holofoil"]; f.ProductID != 219333 || f.MarketPrice != 312.4 {
		t.Errorf("unlimited tcgplayer holofoil finish = %+v", f)
	}
	if _, ok := unlimited.TCGPlayer.Finishes["unit"]; ok {
		t.Error("tcgplayer finishes should not include the \"unit\" metadata key")
	}
	// ...but the 1st Edition Shadowless printing does not, and must not inherit
	// the Unlimited printing's price.
	if stamped := card.Variants[1].Pricing; stamped.Cardmarket.IDProduct != 0 || stamped.Cardmarket.Avg != 0 {
		t.Errorf("stamped shadowless pricing = %+v, want zero value", stamped)
	}
}

// The client must not fall back to GraphQL for any language: one transport is the
// point of the package, and a second one is a place for the two to disagree about
// which cards exist.
func TestNoTransportOtherThanREST(t *testing.T) {
	for _, lang := range []string{"en", "ja"} {
		t.Run(lang, func(t *testing.T) {
			var mu sync.Mutex
			var methods, paths []string
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				methods, paths = append(methods, r.Method), append(paths, r.URL.Path)
				mu.Unlock()
				if strings.Count(r.URL.Path, "/") > 2 {
					w.Write([]byte(`{"id":"base1-4","localId":"4","name":"Charizard","set":{"id":"base1"}}`))
					return
				}
				w.Write([]byte(`[{"id":"base1-4"}]`))
			})

			if _, err := c.CardsByDex(context.Background(), lang, 6); err != nil {
				t.Fatalf("CardsByDex: %v", err)
			}
			for i, p := range paths {
				if strings.Contains(p, "graphql") {
					t.Errorf("hit %q, want a REST path", p)
				}
				if !strings.HasPrefix(p, "/"+lang+"/") {
					t.Errorf("path %q is not under /%s/", p, lang)
				}
				if methods[i] != http.MethodGet {
					t.Errorf("path %q used %s, want GET", p, methods[i])
				}
			}
		})
	}
}

// A rarity of "None" is tcgdex's way of saying "no rarity", not a rarity called
// None — it must not reach the product as a printable label.
func TestRarityNoneBecomesEmpty(t *testing.T) {
	c, _ := restServer(t, `[{"id":"2024sv-1"}]`, map[string]string{
		"2024sv-1": `{"id":"2024sv-1","localId":"1","name":"Charizard","rarity":"None","image":null,
		  "set":{"id":"2024sv","name":"McDonald's Collection 2024","cardCount":{"official":15,"total":15}}}`,
	})

	got, err := c.CardsByDex(context.Background(), "en", 6)
	if err != nil {
		t.Fatalf("CardsByDex: %v", err)
	}
	if got[0].Rarity != "" {
		t.Errorf("Rarity = %q, want empty", got[0].Rarity)
	}
	if got[0].ImageBase != "" {
		t.Errorf("ImageBase = %q, want empty for a null image", got[0].ImageBase)
	}
}

func TestCardsByDexJapanese(t *testing.T) {
	var listPath, detailPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ja/cards":
			listPath = r.URL.String()
			w.Write([]byte(`[{"id":"PMCG1-021"}]`))
		default:
			detailPath = r.URL.Path
			w.Write([]byte(`{"id":"PMCG1-021","name":"リザードン","localId":"021","dexId":[6],
			  "rarity":"Holo Rare","illustrator":"Mitsuhiro Arita",
			  "variants_detailed":[{"type":"holo","size":"standard"},{"type":"holo","subtype":"no-rarity","size":"standard"}],
			  "set":{"id":"PMCG1","name":"拡張パック","cardCount":{"official":102,"total":102}}}`))
		}
	})

	got, err := c.CardsByDex(context.Background(), "ja", 6)
	if err != nil {
		t.Fatalf("CardsByDex: %v", err)
	}
	if listPath != "/ja/cards?dexId=eq:6" {
		t.Errorf("list path = %q", listPath)
	}
	if detailPath != "/ja/cards/PMCG1-021" {
		t.Errorf("detail path = %q", detailPath)
	}
	if len(got) != 1 || got[0].Name != "リザードン" {
		t.Fatalf("got %+v", got)
	}
	if len(got[0].Variants) != 2 {
		t.Errorf("got %d variants, want 2", len(got[0].Variants))
	}
}

// Detail responses arrive concurrently, so the client has to put them back in list
// order rather than in the order the network happened to answer.
func TestCardOrderFollowsTheList(t *testing.T) {
	ids := []string{"base1-4", "base2-4", "base3-4", "gym2-2", "neo4-107"}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if id := strings.TrimPrefix(r.URL.Path, "/en/cards/"); id != r.URL.Path {
			// Answer the head of the list last, so a client that appended results as
			// they arrived would visibly reorder them.
			if id == ids[0] {
				time.Sleep(20 * time.Millisecond)
			}
			w.Write([]byte(`{"id":"` + id + `","set":{"id":"base1"}}`))
			return
		}
		w.Write([]byte(`[{"id":"` + strings.Join(ids, `"},{"id":"`) + `"}]`))
	})

	got, err := c.CardsByDex(context.Background(), "en", 6)
	if err != nil {
		t.Fatalf("CardsByDex: %v", err)
	}
	if len(got) != len(ids) {
		t.Fatalf("got %d cards, want %d", len(got), len(ids))
	}
	for i, want := range ids {
		if got[i].ID != want {
			t.Errorf("card %d = %q, want %q", i, got[i].ID, want)
		}
	}
}

func TestRetriesTransientFailures(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/en/cards/") {
			w.Write([]byte(charizardDetail))
			return
		}
		w.Write([]byte(`[{"id":"base1-4"}]`))
	})

	got, err := c.CardsByDex(context.Background(), "en", 6)
	if err != nil {
		t.Fatalf("CardsByDex: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d cards, want 1", len(got))
	}
	// Two failed list attempts, the third succeeding, then one detail request.
	if n := calls.Load(); n != 4 {
		t.Errorf("made %d calls, want 4", n)
	}
}

// A 4xx is a permanent answer; retrying it just makes noise.
func TestDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	})

	if _, err := c.CardsByDex(context.Background(), "en", 6); err == nil {
		t.Fatal("want error")
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("made %d calls, want 1", n)
	}
}

// A Pokémon with no cards in a language is an empty list, not a failure — the UI
// needs to say "no Japanese cards" rather than "something went wrong".
func TestMissingListIsEmptyNotError(t *testing.T) {
	for _, lang := range []string{"en", "ja"} {
		t.Run(lang, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			})

			got, err := c.CardsByDex(context.Background(), lang, 6)
			if err != nil {
				t.Fatalf("CardsByDex: %v", err)
			}
			if len(got) != 0 {
				t.Errorf("got %d cards, want 0", len(got))
			}
		})
	}
}

// A missing *detail* is different: the list said the card exists, so failing to
// read it means the answer would be silently short a card.
func TestMissingDetailIsAnError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/en/cards/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`[{"id":"base1-4"}]`))
	})

	if _, err := c.CardsByDex(context.Background(), "en", 6); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// A cancelled request must surface as context.Canceled rather than being
// retried three times and reported as an upstream failure.
func TestContextCancellationStopsWork(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := c.CardsByDex(ctx, "en", 6); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if n := calls.Load(); n > 1 {
		t.Errorf("made %d calls after cancellation, want at most 1", n)
	}
}

// Release dates only exist on the set detail, so Sets has to read every set
// individually — in both languages, since there is no shortcut for either now.
func TestSetsReadsEveryDetailForReleaseDates(t *testing.T) {
	for _, lang := range []string{"en", "ja"} {
		t.Run(lang, func(t *testing.T) {
			var mu sync.Mutex
			asked := map[string]int{}
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				asked[r.URL.Path]++
				mu.Unlock()

				switch r.URL.Path {
				case "/" + lang + "/sets":
					// The brief list shape: no releaseDate, which is why details follow.
					w.Write([]byte(`[{"id":"base1","name":"Base Set","cardCount":{"official":102,"total":102}},
					                 {"id":"basep","name":"Wizards Black Star Promos","cardCount":{"official":53,"total":53}}]`))
				case "/" + lang + "/sets/base1":
					w.Write([]byte(`{"id":"base1","name":"Base Set","releaseDate":"1999-01-09","serie":{"id":"base","name":"Base"},"cardCount":{"official":102,"total":102}}`))
				default:
					// A promo set with no known release date is a real state, not a fault.
					w.Write([]byte(`{"id":"basep","name":"Wizards Black Star Promos","serie":{"id":"base","name":"Base"},"cardCount":{"official":53,"total":53}}`))
				}
			})

			got, err := c.Sets(context.Background(), lang)
			if err != nil {
				t.Fatalf("Sets: %v", err)
			}
			if len(got) != 2 {
				t.Fatalf("got %d sets, want 2", len(got))
			}
			if got[0].ReleaseDate != "1999-01-09" || got[0].Series != "Base" || got[0].SeriesID != "base" || got[0].CountOfficial != 102 {
				t.Errorf("sets[0] = %+v", got[0])
			}
			if got[1].ReleaseDate != "" {
				t.Errorf("sets[1].ReleaseDate = %q, want empty", got[1].ReleaseDate)
			}
			for _, p := range []string{"/" + lang + "/sets", "/" + lang + "/sets/base1", "/" + lang + "/sets/basep"} {
				if asked[p] != 1 {
					t.Errorf("asked for %s %d times, want 1", p, asked[p])
				}
			}
		})
	}
}
