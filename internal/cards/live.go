package cards

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/ExtraBB/pkmn-master-set/internal/tcgdex"
)

// DefaultTTL is how long a Pokémon's card list is reused before refetching.
// The corpus changes when a new set ships — roughly monthly — so a day is
// generous and still bounds staleness.
const DefaultTTL = 24 * time.Hour

// LiveSource fetches from tcgdex and caches in memory.
//
// Card lists expire; set metadata does not, because a set's name, series and
// release date are fixed once it has shipped.
type LiveSource struct {
	client *tcgdex.Client
	ttl    time.Duration

	mu    sync.Mutex
	cards map[cacheKey]*cardEntry
	sets  map[cacheKey]Set
}

type cacheKey struct {
	lang Language
	id   string
}

type cardEntry struct {
	// done is closed when the first fetch for this key finishes. Waiters block on
	// it instead of issuing their own request, so a burst of visitors asking for
	// Charizard produces one upstream call.
	done    chan struct{}
	cards   []Card
	err     error
	fetched time.Time
}

func NewLiveSource(client *tcgdex.Client, ttl time.Duration) *LiveSource {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &LiveSource{
		client: client,
		ttl:    ttl,
		cards:  make(map[cacheKey]*cardEntry),
		sets:   make(map[cacheKey]Set),
	}
}

// Cards returns a Pokémon's cards, fetching only when the cache is cold or stale.
//
// If a refresh fails but a stale copy exists, the stale copy is served. For this
// product a slightly old complete list is far more useful than an error page —
// the collector is trying to find out what exists, and last week's answer is
// almost always still right.
func (s *LiveSource) Cards(ctx context.Context, lang Language, dexID int) ([]Card, error) {
	key := cacheKey{lang, strconv.Itoa(dexID)}

	s.mu.Lock()
	entry, ok := s.cards[key]
	if ok {
		select {
		case <-entry.done:
			// A completed, fresh entry is a straight hit.
			if entry.err == nil && time.Since(entry.fetched) < s.ttl {
				s.mu.Unlock()
				return entry.cards, nil
			}
			// Stale: refetch, keeping the old data as a fallback.
			stale := entry
			fresh := &cardEntry{done: make(chan struct{})}
			s.cards[key] = fresh
			s.mu.Unlock()
			s.fetchCards(ctx, key, lang, dexID, fresh, stale)
			return fresh.cards, fresh.err
		default:
			// Someone else is already fetching; wait for their result rather than
			// issuing a duplicate request.
			s.mu.Unlock()
			select {
			case <-entry.done:
				return entry.cards, entry.err
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	fresh := &cardEntry{done: make(chan struct{})}
	s.cards[key] = fresh
	s.mu.Unlock()
	s.fetchCards(ctx, key, lang, dexID, fresh, nil)
	return fresh.cards, fresh.err
}

func (s *LiveSource) fetchCards(ctx context.Context, key cacheKey, lang Language, dexID int, entry, stale *cardEntry) {
	defer close(entry.done)

	raw, err := s.client.CardsByDex(ctx, string(lang), dexID)
	if err != nil {
		if stale != nil && stale.err == nil {
			entry.cards, entry.fetched = stale.cards, stale.fetched
			return
		}
		entry.err = err
		// A failed entry is not worth keeping: leaving it in place would make
		// every later request wait out the TTL before retrying.
		s.mu.Lock()
		if s.cards[key] == entry {
			delete(s.cards, key)
		}
		s.mu.Unlock()
		return
	}

	entry.cards = make([]Card, len(raw))
	for i, c := range raw {
		entry.cards[i] = fromWire(c)
	}
	entry.fetched = time.Now()
}

// Set returns set metadata, cached indefinitely.
func (s *LiveSource) Set(ctx context.Context, lang Language, setID string) (Set, error) {
	key := cacheKey{lang, setID}

	s.mu.Lock()
	if set, ok := s.sets[key]; ok {
		s.mu.Unlock()
		return set, nil
	}
	s.mu.Unlock()

	raw, err := s.client.SetDetail(ctx, string(lang), setID)
	if err != nil {
		return Set{}, err
	}
	date, err := ParseDate(raw.ReleaseDate)
	if err != nil {
		// An unparseable date is treated as unknown rather than as a failure: the
		// set and its cards still exist, they just sort to the end.
		date = Date{}
	}
	set := Set{
		ID:            raw.ID,
		Name:          raw.Name,
		SeriesID:      raw.SeriesID,
		Series:        raw.Series,
		ReleaseDate:   date,
		CountOfficial: raw.CountOfficial,
	}

	s.mu.Lock()
	s.sets[key] = set
	s.mu.Unlock()
	return set, nil
}

// WarmSets preloads the whole set index for a language in one pass, so the first
// visitor does not pay for a set lookup per set in their Pokémon's list.
func (s *LiveSource) WarmSets(ctx context.Context, lang Language) error {
	raw, err := s.client.Sets(ctx, string(lang))
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range raw {
		date, _ := ParseDate(r.ReleaseDate)
		s.sets[cacheKey{lang, r.ID}] = Set{
			ID:            r.ID,
			Name:          r.Name,
			SeriesID:      r.SeriesID,
			Series:        r.Series,
			ReleaseDate:   date,
			CountOfficial: r.CountOfficial,
		}
	}
	return nil
}

func fromWire(c tcgdex.Card) Card {
	variants := make([]Variant, len(c.Variants))
	for i, v := range c.Variants {
		variants[i] = Variant{
			Type: v.Type, Subtype: v.Subtype, Stamps: v.Stamp, Size: v.Size,
			Pricing: fromWirePricing(v.Pricing),
		}
	}
	return Card{
		ID:          c.ID,
		Name:        c.Name,
		Number:      c.Number,
		DexIDs:      c.DexIDs,
		Rarity:      c.Rarity,
		Illustrator: c.Illustrator,
		ImageBase:   c.ImageBase,
		SetID:       c.SetID,
		SetName:     c.SetName,
		SetTotal:    c.SetTotal,
		Variants:    NormalizeVariants(variants),
		Pricing:     fromWirePricing(c.Pricing),
	}
}

func fromWirePricing(p tcgdex.Pricing) Pricing {
	out := Pricing{
		Cardmarket: CardmarketPricing{
			ProductID: p.Cardmarket.IDProduct,
			Avg:       p.Cardmarket.Avg,
			AvgHolo:   p.Cardmarket.AvgHolo,
		},
	}
	if len(p.TCGPlayer.Finishes) == 0 {
		return out
	}
	out.TCGPlayer.Finishes = make(map[string]TCGPlayerFinish, len(p.TCGPlayer.Finishes))
	for key, f := range p.TCGPlayer.Finishes {
		out.TCGPlayer.Finishes[key] = TCGPlayerFinish{ProductID: f.ProductID, MarketPrice: f.MarketPrice}
	}
	return out
}
