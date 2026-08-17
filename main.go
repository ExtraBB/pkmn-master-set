package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ExtraBB/pkmn-master-set/internal/cards"
	"github.com/ExtraBB/pkmn-master-set/internal/server"
	"github.com/ExtraBB/pkmn-master-set/internal/tcgdex"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	species, err := cards.EmbeddedSpecies()
	if err != nil {
		log.Fatalf("species index: %v", err)
	}

	source := cards.NewLiveSource(tcgdex.New(), cacheTTL())
	catalog := cards.NewCatalog(source, species)

	// Warm the set index in the background. It gives the first visitor release
	// dates without a lookup per set, but the server must still start when the
	// upstream API is unreachable — set metadata is filled in lazily either way.
	go warmSets(source)

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.New(catalog).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// cacheTTL is how long a fetched card list is reused. Overridable with
// PKMN_CACHE_TTL (any Go duration, e.g. "1h").
func cacheTTL() time.Duration {
	raw := os.Getenv("PKMN_CACHE_TTL")
	if raw == "" {
		return cards.DefaultTTL
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("PKMN_CACHE_TTL=%q is not a duration, using %s", raw, cards.DefaultTTL)
		return cards.DefaultTTL
	}
	return ttl
}

func warmSets(source *cards.LiveSource) {
	// English comes from a single request; Japanese needs one per set, so it is
	// left to fill in on demand rather than firing ~180 requests at boot.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := source.WarmSets(ctx, cards.LangEN); err != nil {
		log.Printf("warming the English set index: %v (release dates will load on demand)", err)
		return
	}
	log.Print("English set index warmed")
}
