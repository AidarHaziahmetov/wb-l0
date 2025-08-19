package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"wb-l0-go/internal/cache"
	"wb-l0-go/internal/domain"
	"wb-l0-go/internal/repository"
)

type OrderService struct {
	repo  repository.OrderRepository
	cache cache.Cache
	log   *zap.Logger
	pool  *pgxpool.Pool
}

func NewOrderService(repo repository.OrderRepository, cache cache.Cache, log *zap.Logger, pool *pgxpool.Pool) *OrderService {
	return &OrderService{repo: repo, cache: cache, log: log, pool: pool}
}

func (s *OrderService) HandleKafkaOrder(ctx context.Context, key string, payload []byte) error {
	var msg domain.Order
	if err := json.Unmarshal(payload, &msg); err != nil {
		s.log.Error("failed to unmarshal order", zap.Error(err))
		return err
	}
	// Попробуем заполнить order_uid ключом, если он пуст и ключ задан
	if msg.OrderUID == "" && key != "" {
		msg.OrderUID = key
	}

	// Валидация заказа перед сохранением
	if err := s.Validate(msg); err != nil {
		s.log.Error("order validation failed", zap.String("order_uid", msg.OrderUID), zap.Error(err))
		return fmt.Errorf("order validation failed: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.log.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Сохраняем заказ(транзакция не обязательна, тк делаю вставку только в одну таблицу, но решил оставить)
	if err := s.repo.SaveWithTx(ctx, tx, msg); err != nil {
		s.log.Error("failed to save order", zap.String("order_uid", msg.OrderUID), zap.Error(err))
		return fmt.Errorf("failed to save order: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		s.log.Error("failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	// Кэшируем заказ для быстрого доступа
	s.cache.Put(ctx, msg)
	s.log.Debug("order stored", zap.String("order_uid", msg.OrderUID), zap.Int("payload_len", len(payload)))
	return nil
}

func (s *OrderService) ListOrdersUIDs(ctx context.Context, limit, offset int) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListUIDs(ctx, limit, offset)
}

func (s *OrderService) ListOrders(ctx context.Context, limit, offset int) ([]domain.Order, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}

func (s *OrderService) GetOrder(ctx context.Context, orderUID string) (domain.Order, error) {
	if order, ok := s.cache.Get(ctx, orderUID); ok {
		return order, nil
	}
	order, err := s.repo.Get(ctx, orderUID)
	if err != nil {
		return domain.Order{}, err
	}
	s.cache.Put(ctx, order)
	return order, nil
}

// Validate проверяет соответствие заказа требуемой структуре
func (s *OrderService) Validate(order domain.Order) error {
	var errors []string

	// Проверка обязательных полей верхнего уровня
	if strings.TrimSpace(order.OrderUID) == "" {
		errors = append(errors, "order_uid is required")
	}
	if strings.TrimSpace(order.TrackNumber) == "" {
		errors = append(errors, "track_number is required")
	}
	if strings.TrimSpace(order.Entry) == "" {
		errors = append(errors, "entry is required")
	}
	if strings.TrimSpace(order.Locale) == "" {
		errors = append(errors, "locale is required")
	}
	if strings.TrimSpace(order.CustomerID) == "" {
		errors = append(errors, "customer_id is required")
	}
	if strings.TrimSpace(order.DeliveryService) == "" {
		errors = append(errors, "delivery_service is required")
	}
	if strings.TrimSpace(order.ShardKey) == "" {
		errors = append(errors, "shardkey is required")
	}
	if order.SmID == 0 {
		errors = append(errors, "sm_id is required")
	}
	if order.DateCreated.IsZero() {
		errors = append(errors, "date_created is required")
	}
	if strings.TrimSpace(order.OofShard) == "" {
		errors = append(errors, "oof_shard is required")
	}

	// Проверка delivery
	if err := s.validateDelivery(order.Delivery); err != nil {
		errors = append(errors, fmt.Sprintf("delivery: %s", err))
	}

	// Проверка payment
	if err := s.validatePayment(order.Payment); err != nil {
		errors = append(errors, fmt.Sprintf("payment: %s", err))
	}

	// Проверка items
	if err := s.validateItems(order.Items); err != nil {
		errors = append(errors, fmt.Sprintf("items: %s", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateDelivery проверяет структуру delivery
func (s *OrderService) validateDelivery(delivery domain.Delivery) error {
	var errors []string

	if strings.TrimSpace(delivery.Name) == "" {
		errors = append(errors, "name is required")
	}
	if strings.TrimSpace(delivery.Phone) == "" {
		errors = append(errors, "phone is required")
	}
	if strings.TrimSpace(delivery.Zip) == "" {
		errors = append(errors, "zip is required")
	}
	if strings.TrimSpace(delivery.City) == "" {
		errors = append(errors, "city is required")
	}
	if strings.TrimSpace(delivery.Address) == "" {
		errors = append(errors, "address is required")
	}
	if strings.TrimSpace(delivery.Region) == "" {
		errors = append(errors, "region is required")
	}
	if strings.TrimSpace(delivery.Email) == "" {
		errors = append(errors, "email is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

// validatePayment проверяет структуру payment
func (s *OrderService) validatePayment(payment domain.Payment) error {
	var errors []string

	if strings.TrimSpace(payment.Transaction) == "" {
		errors = append(errors, "transaction is required")
	}
	if strings.TrimSpace(payment.Currency) == "" {
		errors = append(errors, "currency is required")
	}
	if strings.TrimSpace(payment.Provider) == "" {
		errors = append(errors, "provider is required")
	}
	if payment.Amount == 0 {
		errors = append(errors, "amount is required")
	}
	if payment.PaymentDt == 0 {
		errors = append(errors, "payment_dt is required")
	}
	if strings.TrimSpace(payment.Bank) == "" {
		errors = append(errors, "bank is required")
	}
	if payment.DeliveryCost == 0 {
		errors = append(errors, "delivery_cost is required")
	}
	if payment.GoodsTotal == 0 {
		errors = append(errors, "goods_total is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

// validateItems проверяет массив items
func (s *OrderService) validateItems(items []domain.Items) error {
	if len(items) == 0 {
		return fmt.Errorf("at least one item is required")
	}

	for i, item := range items {
		if err := s.validateItem(item, i); err != nil {
			return err
		}
	}
	return nil
}

// validateItem проверяет отдельный элемент items
func (s *OrderService) validateItem(item domain.Items, index int) error {
	var errors []string

	if item.ChrtID == 0 {
		errors = append(errors, "chrt_id is required")
	}
	if strings.TrimSpace(item.TrackNumber) == "" {
		errors = append(errors, "track_number is required")
	}
	if item.Price == 0 {
		errors = append(errors, "price is required")
	}
	if strings.TrimSpace(item.Rid) == "" {
		errors = append(errors, "rid is required")
	}
	if strings.TrimSpace(item.Name) == "" {
		errors = append(errors, "name is required")
	}
	if item.TotalPrice == 0 {
		errors = append(errors, "total_price is required")
	}
	if item.NmID == 0 {
		errors = append(errors, "nm_id is required")
	}
	if strings.TrimSpace(item.Brand) == "" {
		errors = append(errors, "brand is required")
	}
	if item.Status == 0 {
		errors = append(errors, "status is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("item[%d]: %s", index, strings.Join(errors, "; "))
	}
	return nil
}
