package main

import (
	"context"
	"log"
	"time"

	pb "github.com/ruslan-codebase/brics-currency-exchange/proto"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer conn.Close()

	client := pb.NewCurrencyServiceClient(conn)

	// Test conversion: 1000 RUB → CNY
	req := &pb.ConvertRequest{
		FromCurrency: "RUB",
		ToCurrency:   "CNY",
		Amount:       1000,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.Convert(ctx, req)
	if err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}

	log.Printf("Converted %.2f %s → %.2f %s",
		req.GetAmount(),
		req.GetFromCurrency(),
		res.GetConvertedAmount(),
		req.GetToCurrency())
	log.Printf("Exchange rate: %.4f | Service fee: %.2f %s",
		res.GetExchangeRate(),
		res.GetServiceFee(),
		req.GetFromCurrency())
}
