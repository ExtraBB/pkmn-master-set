// Package tcgdex is a read-only client for the tcgdex.net REST API
// (https://tcgdex.dev/rest/cards).
//
// REST is the only transport used, for every language. The API also exposes a
// GraphQL endpoint, but it is English-only — it ignores Accept-Language and
// /v2/ja/graphql does not exist — so relying on it meant maintaining two code
// paths that had to be kept in agreement about which cards exist. They returned
// the same corpus, so the second path bought nothing but a place for the two to
// drift.
//
// The cost of REST-only is that the list endpoints return a trimmed shape with no
// field selection, so a Pokémon is one list request plus one detail request per
// card, fanned out by forEach. Callers are expected to cache.
//
// Two REST behaviours shape the code below:
//
//   - A card response's nested set object carries a null releaseDate, so sets are
//     always fetched separately and joined by set ID by the caller.
//   - Unrecognised query parameters are treated as filters that match nothing,
//     so a mistyped parameter yields an empty list rather than an error. Every
//     query string here is built from a checked constant for that reason.
package tcgdex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL   = "https://api.tcgdex.net/v2"
	defaultUserAgent = "pkmn-master-set/0.1 (+github.com/ExtraBB/pkmn-master-set)"
	defaultTimeout   = 30 * time.Second

	// detailConcurrency bounds the card-by-card fan-out. Every language pays it
	// now, including English, so it is sized against the largest list in the
	// corpus: Pikachu's ~200 cards take about a second at 16 and two and a half at
	// 8. Still a polite number of connections to hold open against one host.
	detailConcurrency = 16

	maxAttempts = 3
)

// ErrNotFound is returned when the API reports a resource does not exist.
var ErrNotFound = errors.New("tcgdex: not found")

// Client is safe for concurrent use.
type Client struct {
	baseURL   string
	http      *http.Client
	userAgent string
	// backoff returns the pause before the (attempt+1)-th try. Overridable so
	// tests do not sleep.
	backoff func(attempt int) time.Duration
}

type Option func(*Client)

func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimSuffix(u, "/") }
}

func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

func WithUserAgent(s string) Option { return func(c *Client) { c.userAgent = s } }

// WithBackoff overrides the retry pause. Used by tests to retry instantly.
func WithBackoff(f func(attempt int) time.Duration) Option {
	return func(c *Client) { c.backoff = f }
}

func New(opts ...Option) *Client {
	c := &Client{
		baseURL:   defaultBaseURL,
		http:      &http.Client{Timeout: defaultTimeout},
		userAgent: defaultUserAgent,
		backoff:   defaultBackoff,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func defaultBackoff(attempt int) time.Duration {
	base := time.Duration(500<<attempt) * time.Millisecond // 500ms, 1s, 2s
	return base + time.Duration(rand.Int63n(int64(base/2)))
}

// Sets returns the full set index for a language, including release dates.
//
// Set details are the only place release dates are available: the set object
// nested inside a card response carries a null releaseDate.
//
// The set list is a brief shape without release dates, so each set has to be read
// individually — a couple of hundred requests. Callers should treat this as a
// one-off warm-up, not a per-request cost.
func (c *Client) Sets(ctx context.Context, lang string) ([]Set, error) {
	var brief []briefRef
	if err := c.get(ctx, "/"+lang+"/sets", &brief); err != nil {
		return nil, err
	}

	sets := make([]Set, len(brief))
	err := c.forEach(ctx, len(brief), func(ctx context.Context, i int) error {
		s, err := c.SetDetail(ctx, lang, brief[i].ID)
		if err != nil {
			return err
		}
		sets[i] = s
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sets, nil
}

// SetDetail returns one set including its release date.
func (c *Client) SetDetail(ctx context.Context, lang, setID string) (Set, error) {
	var s restSet
	if err := c.get(ctx, "/"+lang+"/sets/"+url.PathEscape(setID), &s); err != nil {
		return Set{}, err
	}
	return s.toSet(), nil
}

// CardsByDex returns every card in a language whose dexId array contains dexID.
//
// The filter is strict equality (`dexId=eq:N`) rather than the API's default laxist
// match, which compares as a substring: `dexId=25` also returns dex 125 and 250.
// Because dexId is an array, strict equality still matches any element, so cards
// featuring two Pokémon (Pikachu & Zekrom GX is dexId [25, 644]) are returned for
// both — which is the attribution rule this product wants.
//
// A card the source has not tagged with a dexId cannot be reached by any query
// here and so appears in nobody's list. That blind spot is measured by
// TestUntaggedPokemonCardsStaySmall rather than papered over.
func (c *Client) CardsByDex(ctx context.Context, lang string, dexID int) ([]Card, error) {
	var brief []briefRef
	path := "/" + lang + "/cards?dexId=eq:" + strconv.Itoa(dexID)
	if err := c.get(ctx, path, &brief); err != nil {
		// A Pokémon with no cards in this language is an empty list, not an error.
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	cards := make([]Card, len(brief))
	err := c.forEach(ctx, len(brief), func(ctx context.Context, i int) error {
		var rc restCard
		if err := c.get(ctx, "/"+lang+"/cards/"+url.PathEscape(brief[i].ID), &rc); err != nil {
			return err
		}
		cards[i] = rc.toCard()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return cards, nil
}

// forEach runs fn for indices [0,n) with bounded concurrency, returning the first
// error and cancelling the rest. Results are written by index, so the caller's
// slice stays deterministically ordered.
func (c *Client) forEach(ctx context.Context, n int, fn func(context.Context, int) error) error {
	if n == 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, detailConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i := range n {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			mu.Lock()
			defer mu.Unlock()
			if firstErr != nil {
				return firstErr
			}
			return ctx.Err()
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(ctx, i); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
					cancel()
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	return firstErr
}

// ---- transport ----

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		return req, nil
	}, func(body []byte) error {
		return json.Unmarshal(body, out)
	})
}

// do issues the request with retries, then hands the body to decode.
//
// Retries cover transient failures only — 5xx, 429 and transport errors. A 4xx is
// a permanent answer and retrying it would just be noise.
func (c *Client) do(ctx context.Context, mkReq func() (*http.Request, error), decode func([]byte) error) error {
	var lastErr error

	for attempt := range maxAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.backoff(attempt - 1)):
			}
		}

		req, err := mkReq()
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("%w: %s", ErrNotFound, req.URL.Path)
		}
		if retriable(resp.StatusCode) {
			lastErr = fmt.Errorf("tcgdex: %s: %s", req.URL.Path, resp.Status)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("tcgdex: %s: %s", req.URL.Path, resp.Status)
		}
		if readErr != nil {
			lastErr = readErr
			continue
		}
		return decode(body)
	}

	return fmt.Errorf("tcgdex: giving up after %d attempts: %w", maxAttempts, lastErr)
}

func retriable(status int) bool {
	return status >= 500 || status == http.StatusTooManyRequests
}
