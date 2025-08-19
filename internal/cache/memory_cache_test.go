package cache_test

import (
	"context"
	"testing"

	"wb-l0-go/internal/cache"
	"wb-l0-go/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestMemoryCache_Put(t *testing.T) {
	cache := cache.NewMemoryCache(10)

	order := domain.Order{
		OrderUID: "123",
	}
	cache.Put(context.Background(), order)

	// Проверяем, что заказ можно получить через публичный метод Get
	retrievedOrder, exists := cache.Get(context.Background(), "123")
	assert.True(t, exists)
	assert.Equal(t, order, retrievedOrder)

	// Проверяем, что несуществующий заказ не возвращается
	_, exists = cache.Get(context.Background(), "nonexistent")
	assert.False(t, exists)
}
