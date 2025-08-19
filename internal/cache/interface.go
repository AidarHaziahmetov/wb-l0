package cache

import (
	"context"
	"wb-l0-go/internal/domain"
)

type Cache interface {
	Put(ctx context.Context, msg domain.Order)
	Get(ctx context.Context, orderUID string) (domain.Order, bool)
}

// OrderLister описывает зависимость, необходимую для предзагрузки кеша
// при старте приложения. Реализация должна уметь возвращать список заказов.
type OrderLister interface {
	ListOrders(ctx context.Context, limit, offset int) ([]domain.Order, error)
}
