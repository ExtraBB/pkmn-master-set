package cards

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ExtraBB/pkmn-master-set/internal/tcgdex"
)

const oneCharizard = `{"data":{"cards":[{"id":"base1-4","localId":"4","name":"Charizard","rarity":"Rare Holo",
  "illustrator":"Mitsuhiro Arita","image":"https://assets.tcgdex.net/en/base/base1/4",
  "variants_detailed":[{"type":"holo","subtype":"unlimited","size":"standard"}],
  "set":{"id":"base1","name":"Base Set","cardCount":{"official":102,"total":102}}}]}}`

func liveSource(t *testing.T, ttl time.Duration, h http.HandlerFunc) *LiveSource {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	client := tcgdex.New(tcgdex.WithBaseURL(srv.URL), tcgdex.WithBackoff(func(int) time.Duration { return 0 }))
	return NewLiveSource(client, ttl)
}

func TestCachedCardsAreNotRefetched(t *testing.T) {
	var calls atomic.Int32
	src := liveSource(t, time.Hour, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(oneCharizard))
	})

	for range 3 {
		if _, err := src.Cards(context.Background(), LangEN, 6); err != nil {
			t.Fatalf("Cards: %v", err)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("made %d upstream calls, want 1", n)
	}
}

// A burst of visitors asking for the same Pokémon should produce one upstream
// request, not one per visitor.
func TestConcurrentRequestsShareOneFetch(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	src := liveSource(t, time.Hour, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		<-release
		w.Write([]byte(oneCharizard))
	})

	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = src.Cards(context.Background(), LangEN, 6)
		}()
	}

	// Give the goroutines a moment to pile up on the in-flight entry.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("caller %d: %v", i, err)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("made %d upstream calls for 10 concurrent readers, want 1", n)
	}
}

func TestStaleEntriesAreRefetched(t *testing.T) {
	var calls atomic.Int32
	src := liveSource(t, time.Nanosecond, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(oneCharizard))
	})

	for range 2 {
		if _, err := src.Cards(context.Background(), LangEN, 6); err != nil {
			t.Fatalf("Cards: %v", err)
		}
		time.Sleep(time.Millisecond)
	}
	if n := calls.Load(); n != 2 {
		t.Errorf("made %d upstream calls, want 2", n)
	}
}

// A slightly old but complete list beats an error page: the collector is trying
// to find out what exists, and last week's answer is almost always still right.
func TestStaleDataIsServedWhenRefreshFails(t *testing.T) {
	var fail atomic.Bool
	src := liveSource(t, time.Nanosecond, func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(oneCharizard))
	})

	if _, err := src.Cards(context.Background(), LangEN, 6); err != nil {
		t.Fatalf("warming the cache: %v", err)
	}

	fail.Store(true)
	time.Sleep(time.Millisecond)

	got, err := src.Cards(context.Background(), LangEN, 6)
	if err != nil {
		t.Fatalf("want the stale copy, got error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "base1-4" {
		t.Errorf("got %+v, want the stale Charizard", got)
	}
}

// A failure must not be cached: the next visitor should get a real retry rather
// than wait out the TTL.
func TestFailuresAreNotCached(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	var calls atomic.Int32
	src := liveSource(t, time.Hour, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(oneCharizard))
	})

	if _, err := src.Cards(context.Background(), LangEN, 6); err == nil {
		t.Fatal("want an error on the first attempt")
	}

	fail.Store(false)
	got, err := src.Cards(context.Background(), LangEN, 6)
	if err != nil {
		t.Fatalf("second attempt should retry: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d cards, want 1", len(got))
	}
}

// Set metadata is immutable once a set has shipped, so it never expires.
func TestSetsAreCachedIndefinitely(t *testing.T) {
	var calls atomic.Int32
	src := liveSource(t, time.Nanosecond, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"id":"base1","name":"Base Set","releaseDate":"1999-01-09","serie":{"name":"Base"},"cardCount":{"official":102,"total":102}}`))
	})

	for range 3 {
		got, err := src.Set(context.Background(), LangEN, "base1")
		if err != nil {
			t.Fatalf("Set: %v", err)
		}
		if got.ReleaseDate.String() != "1999-01-09" || got.Series != "Base" {
			t.Fatalf("set = %+v", got)
		}
		time.Sleep(time.Millisecond)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("made %d upstream calls, want 1", n)
	}
}

// Warming the set index up front saves the first visitor a lookup per set in
// their Pokémon's list.
func TestWarmSetsPopulatesTheCache(t *testing.T) {
	var calls atomic.Int32
	src := liveSource(t, time.Hour, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"data":{"sets":[{"id":"base1","name":"Base Set","releaseDate":"1999-01-09","serie":{"name":"Base"},"cardCount":{"official":102,"total":102}}]}}`))
	})

	if err := src.WarmSets(context.Background(), LangEN); err != nil {
		t.Fatalf("WarmSets: %v", err)
	}
	got, err := src.Set(context.Background(), LangEN, "base1")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got.Name != "Base Set" {
		t.Errorf("set = %+v", got)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("made %d upstream calls, want 1", n)
	}
}
