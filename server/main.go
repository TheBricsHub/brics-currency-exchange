package main

import (
	"context"
	"log"
	"net"

	pb "github.com/ruslan-codebase/brics-currency-exchange/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedCurrencyServiceServer
}

// BRICS exchange rates (simulated)
var rates = map[string]map[string]float64{
	"RUB": {"CNY": 0.079, "INR": 0.93, "BRL": 0.069, "ZAR": 0.19},
	"CNY": {"RUB": 12.66, "INR": 11.76, "BRL": 0.87, "ZAR": 2.41},
	"INR": {"RUB": 1.07, "CNY": 0.085, "BRL": 0.074, "ZAR": 0.20},
	"BRL": {"RUB": 14.48, "CNY": 1.15, "INR": 13.51, "ZAR": 2.75},
	"ZAR": {"RUB": 5.26, "CNY": 0.41, "INR": 4.91, "BRL": 0.36},
}

func (s *server) Convert(ctx context.Context, req *pb.ConvertRequest) (*pb.ConvertResponse, error) {
	from := req.GetFromCurrency()
	to := req.GetToCurrency()
	amount := req.GetAmount()

	log.Printf("Conversion request: %.2f %s => %s", amount, from, to)

	// Get exchange rate
	rate, ok := rates[from][to]
	if !ok {
		return nil, grpc.Errorf(3, "Unsupported currency pair: %s to %s", from, to)
	}

	// Calculate conversion
	fee := amount * 0.001 // 0.1% service fee
	converted := (amount - fee) * rate

	return &pb.ConvertResponse{
		ConvertedAmount: converted,
		ExchangeRate:    rate,
		ServiceFee:      fee,
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterCurrencyServiceServer(s, &server{})

	log.Println("Server started on port 50051")
	log.Println("Supported BRICS currencies: RUB, CNY, INR, BRL, ZAR")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
