package tcgdex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

const charizardGraphQL = `{"data":{"cards":[{
  "id":"base1-4","localId":"4","name":"Charizard","rarity":"Rare","illustrator":"Mitsuhiro Arita",
  "image":"https://assets.tcgdex.net/en/base/base1/4",
  "variants_detailed":[
    {"type":"holo","subtype":"unlimited","size":"standard"},
    {"type":"holo","subtype":"shadowless","stamp":["1st-edition"],"size":"standard"},
    {"type":"holo","subtype":"shadowless","size":"standard"}],
  "set":{"id":"base1","name":"Base Set","cardCount":{"official":102,"total":102}}
}]}}`

func TestCardsByDexEnglishUsesGraphQL(t *testing.T) {
	var path string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write([]byte(charizardGraphQL))
	})

	got, err := c.CardsByDex(context.Background(), "en", 6)
	if err != nil {
		t.Fatalf("CardsByDex: %v", err)
	}
	if path != "/graphql" {
		t.Errorf("path = %q, want /graphql", path)
	}
	if len(got) != 1 {
		t.Fatalf("got %d cards, want 1", len(got))
	}
	card := got[0]
	if card.ID != "base1-4" || card.Number != "4" || card.SetTotal != 102 {
		t.Errorf("card = %+v", card)
	}
	if len(card.Variants) != 3 {
		t.Errorf("got %d variants, want 3", len(card.Variants))
	}
}

// A rarity of "None" is tcgdex's way of saying "no rarity", not a rarity called
// None — it must not reach the product as a printable label.
func TestRarityNoneBecomesEmpty(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":{"cards":[{"id":"2024sv-1","localId":"1","name":"Charizard","rarity":"None","image":null,"set":{"id":"2024sv","name":"McDonald's Collection 2024","cardCount":{"official":15,"total":15}}}]}}`))
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

// GraphQL reports failure inside a 200 response, so a 200 alone must never be
// treated as success.
func TestGraphQLErrorsInOKResponseAreFailures(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"errors":[{"message":"Unknown argument \"lang\""}]}`))
	})

	_, err := c.CardsByDex(context.Background(), "en", 6)
	var gqlErr *GraphQLError
	if !errors.As(err, &gqlErr) {
		t.Fatalf("err = %v, want *GraphQLError", err)
	}
}

func TestCardsByDexJapaneseUsesREST(t *testing.T) {
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

func TestRetriesTransientFailures(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(charizardGraphQL))
	})

	got, err := c.CardsByDex(context.Background(), "en", 6)
	if err != nil {
		t.Fatalf("CardsByDex: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d cards, want 1", len(got))
	}
	if n := calls.Load(); n != 3 {
		t.Errorf("made %d calls, want 3", n)
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
func TestMissingJapaneseListIsEmptyNotError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	got, err := c.CardsByDex(context.Background(), "ja", 6)
	if err != nil {
		t.Fatalf("CardsByDex: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d cards, want 0", len(got))
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

func TestSetsEnglishParsesReleaseDates(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":{"sets":[{"id":"base1","name":"Base Set","releaseDate":"1999-01-09","serie":{"name":"Base"},"cardCount":{"official":102,"total":102}}]}}`))
	})

	got, err := c.Sets(context.Background(), "en")
	if err != nil {
		t.Fatalf("Sets: %v", err)
	}
	if len(got) != 1 || got[0].ReleaseDate != "1999-01-09" || got[0].Series != "Base" {
		t.Fatalf("got %+v", got)
	}
}
