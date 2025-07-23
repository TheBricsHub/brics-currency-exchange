package main

import (
	"context"
	"testing"
	"time"

	pb "github.com/TheBricsHub/brics-currency-exchange/services/exchange/proto"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDuplicateDetection(t *testing.T) {
	// Setup in-memory NATS server for tests
	nc, err := nats.Connect(nats.DefaultURL)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer nc.Close()

	js, err := nc.JetStream()
	assert.NoError(t, err)

	// Create test stream
	_, err = js.AddStream(&nats.StreamConfig{
		Name:       "TEST_TRANSACTIONS",
		Subjects:   []string{"test.>"},
		Duplicates: 5 * time.Minute, // Shorter window for tests
	})
	assert.NoError(t, err)
	defer js.DeleteStream("TEST_TRANSACTIONS")

	// Create test server
	srv := &server{js: js}

	// Test request
	req := &pb.ConvertRequest{
		FromCurrency:   "RUB",
		ToCurrency:     "CNY",
		Amount:         100,
		IdempotencyKey: "test123",
	}

	// First request should succeed
	res, err := srv.Convert(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Second request should fail with duplicate error
	_, err = srv.Convert(context.Background(), req)
	if assert.Error(t, err) {
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, statusErr.Code())
	}

	// Different key should succeed
	req.IdempotencyKey = "test456"
	_, err = srv.Convert(context.Background(), req)
	assert.NoError(t, err)
}
