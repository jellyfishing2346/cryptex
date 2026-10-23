package api

import (
	"sync"

	"github.com/jellyfishing2346/cryptex/internal/models"
)

// tradeLog keeps the most recent trades in memory for GET /trades and for
// clients that connect after the trades happened. It is a fixed-size ring
// buffer: once full, the oldest trade is overwritten.
type tradeLog struct {
	mu     sync.RWMutex
	trades []*models.Trade
	next   int
	full   bool
}

func newTradeLog(capacity int) *tradeLog {
	if capacity < 1 {
		capacity = 1
	}
	return &tradeLog{trades: make([]*models.Trade, capacity)}
}

func (l *tradeLog) add(trades ...*models.Trade) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, t := range trades {
		l.trades[l.next] = t
		l.next = (l.next + 1) % len(l.trades)
		if l.next == 0 {
			l.full = true
		}
	}
}

// recent returns up to limit trades, newest first.
func (l *tradeLog) recent(limit int) []*models.Trade {
	l.mu.RLock()
	defer l.mu.RUnlock()

	size := l.next
	if l.full {
		size = len(l.trades)
	}
	if limit <= 0 || limit > size {
		limit = size
	}
	out := make([]*models.Trade, 0, limit)
	for i := 1; i <= limit; i++ {
		idx := (l.next - i + len(l.trades)) % len(l.trades)
		out = append(out, l.trades[idx])
	}
	return out
}

// last returns the most recent trade, or nil if there are none.
func (l *tradeLog) last() *models.Trade {
	recent := l.recent(1)
	if len(recent) == 0 {
		return nil
	}
	return recent[0]
}
