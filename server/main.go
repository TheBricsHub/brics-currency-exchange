package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	pb "github.com/ruslan-codebase/brics-currency-exchange/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type server struct {
	pb.UnimplementedCurrencyServiceServer
	js    nats.JetStreamContext
	rates map[string]map[string]float64
}

func NewServer(js nats.JetStreamContext) *server {
	s := &server{
		js:    js,
		rates: initialRates(),
	}
	return s
}

func initialRates() map[string]map[string]float64 {
	return map[string]map[string]float64{
		"RUB": {"CNY": 0.079, "INR": 0.93, "BRL": 0.069, "ZAR": 0.19},
		"CNY": {"RUB": 12.66, "INR": 11.76, "BRL": 0.87, "ZAR": 2.41},
		"INR": {"RUB": 1.07, "CNY": 0.085, "BRL": 0.074, "ZAR": 0.20},
		"BRL": {"RUB": 14.48, "CNY": 1.15, "INR": 13.51, "ZAR": 2.75},
		"ZAR": {"RUB": 5.26, "CNY": 0.41, "INR": 4.91, "BRL": 0.36},
	}
}

func (s *server) Convert(ctx context.Context, req *pb.ConvertRequest) (*pb.ConvertResponse, error) {
	// Validate idempotency key
	if req.IdempotencyKey == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}

	from := req.GetFromCurrency()
	to := req.GetToCurrency()
	amount := req.GetAmount()

	// Get exchange rate
	rate, ok := s.rates[from][to]
	if !ok {
		log.Printf("Unsupported currency pair: %s to %s, %v", from, to, s.rates)
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Unsupported currency pair: %s to %s", from, to)
	}

	// Calculate conversion
	fee := amount * 0.001 // 0.1% service fee
	converted := (amount - fee) * rate

	// Publish event
	event, _ := proto.Marshal(&pb.Conversion{
		Ts:              uint64(time.Now().UnixNano()),
		From:            req.FromCurrency,
		To:              req.ToCurrency,
		ConvertedAmount: req.Amount,
		ExchangeRate:    rate,
		ServiceFee:      fee,
	})
	s.js.PublishAsync("transactions.currency",
		[]byte(event),
		nats.MsgId(req.IdempotencyKey),
		nats.ExpectStream("TRANSACTIONS"),
	)

	return &pb.ConvertResponse{
		ConvertedAmount: converted,
		ExchangeRate:    rate,
		ServiceFee:      fee,
	}, nil
}

func isDuplicateError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
		strings.Contains(strings.ToLower(err.Error()), "msg id already seen")
}

func createStream(js nats.JetStreamContext, name string, config nats.StreamConfig) error {
	// if exists
	_, err := js.StreamInfo(name)
	if err == nil {
		return nil
	}

	_, err = js.AddStream(&config)
	return err
}

func HandleErrorFatal(msg string, err error) {
	if err != nil {
		log.Fatalf(msg, err)
	}
}

func main() {
	uri := os.Getenv("NATS_URL")

	nc, err := nats.Connect(uri,
		nats.MaxReconnects(5),
		nats.ReconnectWait(2*time.Second),
		nats.Timeout(10*time.Second),
	)
	HandleErrorFatal("NATS connection failed: %v", err)

	defer nc.Close()

	js, err := nc.JetStream()
	HandleErrorFatal("JetStream init failed: %v", err)

	err = createStream(js, "TRANSACTIONS", nats.StreamConfig{
		Name:        "TRANSACTIONS",
		Description: "BRICS currency conversion events",
		Subjects:    []string{"transactions.>"},
		Retention:   nats.InterestPolicy,
		MaxAge:      24 * time.Hour,
		Duplicates:  24 * time.Hour,
		Storage:     nats.FileStorage,
		Replicas:    1,
	})
	HandleErrorFatal("Stream creation failed: %v", err)

	err = createStream(js, "RATES", nats.StreamConfig{
		Name:     "RATES",
		Subjects: []string{"rates.>"},
	})
	HandleErrorFatal("Stream creation failed: %v", err)

	lis, err := net.Listen("tcp", ":50051")
	HandleErrorFatal("Failed to listen: %v", err)

	s := grpc.NewServer()
	pb.RegisterCurrencyServiceServer(s, &server{js: js, rates: initialRates()})

	log.Println("Server started on port 50051")
	log.Println("Supported BRICS currencies: RUB, CNY, INR, BRL, ZAR")

	err = s.Serve(lis)
	HandleErrorFatal("Failed to serve: %v", err)
}
