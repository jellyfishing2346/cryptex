package main

import (
	"log"
	"os"

	"github.com/jellyfishing2346/cryptex/internal/api"
	"github.com/jellyfishing2346/cryptex/internal/matching"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
)

func main() {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)
	router := api.NewServer(book, engine).Router()

	addr := ":" + port()
	log.Printf("starting Cryptex API on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func port() string {
	if value := os.Getenv("PORT"); value != "" {
		return value
	}
	return "8080"
}
