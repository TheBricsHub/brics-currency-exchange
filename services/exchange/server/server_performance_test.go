//go:build performance
// +build performance

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/TheBricsHub/brics-currency-exchange/services/exchange/proto"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	targetRPS   = 30000.0
	testSeconds = 12
	concurrency = 600

	natsURL  = "nats://brics-nats:4222"
	grpcAddr = "currency-service:50051"
)

func TestPerfConversionRPS(t *testing.T) {
	waitFor(t, "nats", func() error {
		_, err := nats.Connect(natsURL)
		return err
	})

	waitFor(t, "gRPC", func() error {
		conn, err := grpc.Dial(grpcAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1<<20)),
		)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	})

	conn, err := grpc.Dial(grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1<<20)),
	)
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewCurrencyServiceClient(conn)

	var success uint64
	ctx, cancel := context.WithTimeout(context.Background(), testSeconds*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, err := client.Convert(ctx, &pb.ConvertRequest{
						FromCurrency:   "RUB",
						ToCurrency:     "CNY",
						Amount:         100,
						IdempotencyKey: fmt.Sprintf("perf-%d-%d", worker, time.Now().UnixNano()),
					})
					if err == nil {
						atomic.AddUint64(&success, 1)
					}
				}
			}
		}(i)
	}
	wg.Wait()

	rps := float64(success) / testSeconds
	t.Logf("Successful conversions: %d  (%.2f RPS)", success, rps)

	if rps < targetRPS {
		t.Fatalf("RPS %.2f < target %.0f", rps, targetRPS)
	}
	t.Logf("✅ RPS %.2f meets target %.0f", rps, targetRPS)
}

func waitFor(t *testing.T, name string, fn func() error) {
	deadline := time.After(30 * time.Second)
	for {
		if err := fn(); err == nil {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("%s not ready within 30 s", name)
		case <-time.After(500 * time.Millisecond):
		}
	}
}
