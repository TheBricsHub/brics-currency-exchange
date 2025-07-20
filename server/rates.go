package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type Updater struct {
	js         nats.JetStreamContext
	rates      map[string]map[string]float64
	mu         *sync.RWMutex
	interval   time.Duration
	currencies []string
}

func New(js nats.JetStreamContext, rates *map[string]map[string]float64, mu *sync.RWMutex, d time.Duration) *Updater {
	return &Updater{
		js:         js,
		rates:      *rates,
		mu:         mu,
		interval:   d,
		currencies: []string{"RUB", "CNY", "INR", "BRL", "ZAR"},
	}
}

// Start blocks until ctx is done.
func (u *Updater) Start(ctx context.Context) error {
	t := time.NewTicker(u.interval)
	defer t.Stop()

	// initial load
	u.refresh()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			u.refresh()
		}
	}
}

func (u *Updater) refresh() {
	u.mu.Lock()
	defer u.mu.Unlock()

	for _, base := range u.currencies {
		for _, target := range u.currencies {
			if base == target {
				continue
			}
			// simulate API call
			newRate := 0.9 + rand.Float64()*0.2 // ±10 % noise
			u.rates[base][target] = newRate

			// publish event
			msg := fmt.Sprintf(`{"base":"%s","target":"%s","rate":%.4f,"ts":"%s"}`,
				base, target, newRate, time.Now().Format(time.RFC3339))
			if _, err := u.js.Publish("rates.update", []byte(msg)); err != nil {
				log.Printf("updater: publish failed: %v", err)
			}
		}
	}
}
