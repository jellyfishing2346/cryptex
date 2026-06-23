// Package nats provides NATS streaming for trade events.
package nats

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/nats-io/nats.go"
)

// Publisher publishes trade events to NATS for downstream consumers.
type Publisher struct {
	nc *nats.Conn
}

// New creates a new NATS publisher.
func New(url string) (*Publisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}
	return &Publisher{nc: nc}, nil
}

// Close closes the NATS connection.
func (p *Publisher) Close() error {
	if p.nc != nil {
		p.nc.Close()
	}
	return nil
}

// PublishTrade publishes a trade event to NATS.
func (p *Publisher) PublishTrade(trade *models.Trade) error {
	if p.nc == nil {
		return fmt.Errorf("nats connection is nil")
	}

	subject := fmt.Sprintf("trades.%s", trade.TradingPair)
	data, err := json.Marshal(trade)
	if err != nil {
		return fmt.Errorf("marshal trade: %w", err)
	}

	if err := p.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("publish trade: %w", err)
	}

	log.Printf("published trade %s to %s", trade.ID, subject)
	return nil
}

// PublishOrder publishes an order event to NATS.
func (p *Publisher) PublishOrder(order *models.Order) error {
	if p.nc == nil {
		return fmt.Errorf("nats connection is nil")
	}

	subject := fmt.Sprintf("orders.%s", order.TradingPair)
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal order: %w", err)
	}

	if err := p.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("publish order: %w", err)
	}

	log.Printf("published order %s to %s", order.ID, subject)
	return nil
}

// TradeEvent represents a trade event with additional metadata.
type TradeEvent struct {
	Trade     *models.Trade `json:"trade"`
	Timestamp time.Time     `json:"timestamp"`
	EventType string        `json:"event_type"`
}

// PublishTradeEvent publishes a structured trade event to NATS.
func (p *Publisher) PublishTradeEvent(event *TradeEvent) error {
	if p.nc == nil {
		return fmt.Errorf("nats connection is nil")
	}

	subject := fmt.Sprintf("trade-events.%s", event.Trade.TradingPair)
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal trade event: %w", err)
	}

	if err := p.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("publish trade event: %w", err)
	}

	log.Printf("published trade event %s to %s", event.Trade.ID, subject)
	return nil
}
