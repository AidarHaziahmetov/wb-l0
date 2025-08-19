package cache

import (
	"context"
	"log"
	"sync"

	"wb-l0-go/internal/domain"
)

type MemoryCache struct {
	mu         sync.RWMutex
	order_uids []string
	ordersMap  map[string]domain.Order
	maxItems   int
}

func NewMemoryCache(maxItems int) *MemoryCache {
	if maxItems <= 0 {
		maxItems = 100
	}
	return &MemoryCache{
		order_uids: make([]string, 0, maxItems),
		ordersMap:  make(map[string]domain.Order, maxItems),
		maxItems:   maxItems,
	}
}

func (c *MemoryCache) Put(_ context.Context, msg domain.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.order_uids = append(c.order_uids, msg.OrderUID)

	if len(c.order_uids) > c.maxItems {
		oldest := c.order_uids[0]
		c.order_uids = c.order_uids[1:]
		delete(c.ordersMap, oldest)
	}
	c.ordersMap[msg.OrderUID] = msg
	log.Printf("Order %s added to cache", msg.OrderUID)
	log.Printf("Orders in cache: %v", c.order_uids)
}

func (c *MemoryCache) Get(ctx context.Context, orderUID string) (domain.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	order, ok := c.ordersMap[orderUID]
	return order, ok
}

func (c *MemoryCache) Load(ctx context.Context, lister OrderLister) {
	orders, err := lister.ListOrders(ctx, c.maxItems, 0)
	if err != nil {
		log.Printf("Error loading orders: %v", err)
		return
	}

	for i := len(orders) - 1; i >= 0; i-- {
		c.Put(ctx, orders[i])
	}
}
