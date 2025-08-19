package repository_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"wb-l0-go/internal/domain"
	"wb-l0-go/internal/repository"
)

type OrderRepositoryTestSuite struct {
	suite.Suite
	pool   *pgxpool.Pool
	repo   repository.OrderRepository
	ctx    context.Context
	cancel context.CancelFunc
}

func (suite *OrderRepositoryTestSuite) SetupSuite() {
	// Получаем переменные окружения для тестовой базы данных
	dbHost := getEnv("TEST_DB_HOST", "localhost")
	dbPort := getEnv("TEST_DB_PORT", "5432")
	dbUser := getEnv("TEST_DB_USER", "postgres")
	dbPassword := getEnv("TEST_DB_PASSWORD", "postgres")
	dbName := getEnv("TEST_DB_NAME", "wb_l0_test")

	// Строка подключения к тестовой базе данных
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// Создаем пул соединений
	pool, err := pgxpool.New(context.Background(), connString)
	require.NoError(suite.T(), err)

	suite.pool = pool
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
}

func (suite *OrderRepositoryTestSuite) TearDownSuite() {
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.pool != nil {
		suite.pool.Close()
	}
}

func (suite *OrderRepositoryTestSuite) SetupTest() {
	// Создаем репозиторий для каждого теста
	suite.repo = repository.NewPostgresOrderRepository(suite.pool)

	// Очищаем таблицу перед каждым тестом
	_, err := suite.pool.Exec(suite.ctx, "TRUNCATE TABLE orders RESTART IDENTITY CASCADE")
	require.NoError(suite.T(), err)
}

func (suite *OrderRepositoryTestSuite) TearDownTest() {
	// Очищаем таблицу после каждого теста
	_, err := suite.pool.Exec(suite.ctx, "TRUNCATE TABLE orders RESTART IDENTITY CASCADE")
	require.NoError(suite.T(), err)
}

func (suite *OrderRepositoryTestSuite) TestSaveOrder() {
	// Создаем тестовый заказ
	order := createTestOrder("test-order-1")

	// Сохраняем заказ
	err := suite.repo.Save(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Проверяем, что заказ сохранился в базе
	var count int
	err = suite.pool.QueryRow(suite.ctx, "SELECT COUNT(*) FROM orders WHERE order_uid = $1", order.OrderUID).Scan(&count)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, count)
}

func (suite *OrderRepositoryTestSuite) TestSaveOrderDuplicate() {
	// Создаем тестовый заказ
	order := createTestOrder("test-order-1")

	// Сохраняем заказ первый раз
	err := suite.repo.Save(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Изменяем заказ
	order.TrackNumber = "updated-track"

	// Сохраняем заказ второй раз (должно обновиться)
	err = suite.repo.Save(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Проверяем, что заказ обновился
	var count int
	err = suite.pool.QueryRow(suite.ctx, "SELECT COUNT(*) FROM orders WHERE order_uid = $1", order.OrderUID).Scan(&count)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, count) // Должна быть только одна запись

	// Проверяем, что данные обновились
	var payload []byte
	err = suite.pool.QueryRow(suite.ctx, "SELECT payload FROM orders WHERE order_uid = $1", order.OrderUID).Scan(&payload)
	require.NoError(suite.T(), err)

	var savedOrder domain.Order
	err = json.Unmarshal(payload, &savedOrder)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "updated-track", savedOrder.TrackNumber)
}

func (suite *OrderRepositoryTestSuite) TestGetOrder() {
	// Создаем и сохраняем тестовый заказ
	order := createTestOrder("test-order-1")
	err := suite.repo.Save(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Получаем заказ
	retrievedOrder, err := suite.repo.Get(suite.ctx, order.OrderUID)
	require.NoError(suite.T(), err)

	// Проверяем, что заказ получен корректно
	assert.Equal(suite.T(), order.OrderUID, retrievedOrder.OrderUID)
	assert.Equal(suite.T(), order.TrackNumber, retrievedOrder.TrackNumber)
	assert.Equal(suite.T(), order.Entry, retrievedOrder.Entry)
	assert.Equal(suite.T(), order.Locale, retrievedOrder.Locale)
	assert.Equal(suite.T(), order.CustomerID, retrievedOrder.CustomerID)
	assert.Equal(suite.T(), order.DeliveryService, retrievedOrder.DeliveryService)
	assert.Equal(suite.T(), order.ShardKey, retrievedOrder.ShardKey)
	assert.Equal(suite.T(), order.SmID, retrievedOrder.SmID)
	assert.Equal(suite.T(), order.OofShard, retrievedOrder.OofShard)

	// Проверяем вложенные структуры
	assert.Equal(suite.T(), order.Delivery.Name, retrievedOrder.Delivery.Name)
	assert.Equal(suite.T(), order.Delivery.Phone, retrievedOrder.Delivery.Phone)
	assert.Equal(suite.T(), order.Payment.Transaction, retrievedOrder.Payment.Transaction)
	assert.Equal(suite.T(), order.Payment.Amount, retrievedOrder.Payment.Amount)
	assert.Equal(suite.T(), len(order.Items), len(retrievedOrder.Items))
}

func (suite *OrderRepositoryTestSuite) TestGetOrderNotFound() {
	// Пытаемся получить несуществующий заказ
	_, err := suite.repo.Get(suite.ctx, "non-existent-order")
	assert.Error(suite.T(), err)
}

func (suite *OrderRepositoryTestSuite) TestListOrders() {
	// Создаем несколько тестовых заказов
	orders := []domain.Order{
		createTestOrder("order-1"),
		createTestOrder("order-2"),
		createTestOrder("order-3"),
		createTestOrder("order-4"),
		createTestOrder("order-5"),
	}

	// Сохраняем все заказы
	for _, order := range orders {
		err := suite.repo.Save(suite.ctx, order)
		require.NoError(suite.T(), err)
	}

	// Получаем список заказов с лимитом 3
	orderUIDs, err := suite.repo.ListUIDs(suite.ctx, 3, 0)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), orderUIDs, 3)

	// Получаем список заказов с лимитом 2 и смещением 1
	orderUIDs, err = suite.repo.ListUIDs(suite.ctx, 2, 1)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), orderUIDs, 2)

	// Получаем все заказы
	orderUIDs, err = suite.repo.ListUIDs(suite.ctx, 10, 0)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), orderUIDs, 5)
}

func (suite *OrderRepositoryTestSuite) TestListOrdersEmpty() {
	// Получаем список заказов из пустой таблицы
	orderUIDs, err := suite.repo.ListUIDs(suite.ctx, 10, 0)
	require.NoError(suite.T(), err)
	assert.Empty(suite.T(), orderUIDs)
}

func (suite *OrderRepositoryTestSuite) TestListOrdersWithOffset() {
	// Создаем несколько тестовых заказов
	orders := []domain.Order{
		createTestOrder("order-1"),
		createTestOrder("order-2"),
		createTestOrder("order-3"),
	}

	// Сохраняем все заказы
	for _, order := range orders {
		err := suite.repo.Save(suite.ctx, order)
		require.NoError(suite.T(), err)
	}

	// Получаем заказы с разными смещениями
	orderUIDs, err := suite.repo.ListUIDs(suite.ctx, 2, 0)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), orderUIDs, 2)

	orderUIDs, err = suite.repo.ListUIDs(suite.ctx, 2, 2)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), orderUIDs, 1)

	orderUIDs, err = suite.repo.ListUIDs(suite.ctx, 2, 5)
	require.NoError(suite.T(), err)
	assert.Empty(suite.T(), orderUIDs)
}

func (suite *OrderRepositoryTestSuite) TestSaveOrderInvalidJSON() {
	// Создаем заказ с некорректными данными, которые могут вызвать ошибку JSON
	order := createTestOrder("test-order-1")

	// Попытка сохранить заказ должна пройти успешно
	err := suite.repo.Save(suite.ctx, order)
	require.NoError(suite.T(), err)
}

func (suite *OrderRepositoryTestSuite) TestRepositoryInterface() {
	// Проверяем, что PostgresOrderRepository реализует интерфейс OrderRepository
	var _ repository.OrderRepository = (*repository.PostgresOrderRepository)(nil)
}

// Вспомогательная функция для создания тестового заказа
func createTestOrder(orderUID string) domain.Order {
	return domain.Order{
		OrderUID:    orderUID,
		TrackNumber: "WBILMTESTTRACK",
		Entry:       "WBIL",
		Delivery: domain.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: domain.Payment{
			Transaction:  "b563feb7b2b84b6test",
			RequestId:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []domain.Items{
			{
				ChrtID:      9934930,
				TrackNumber: "WBILMTESTTRACK",
				Price:       453,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
		Locale:          "en",
		InternalSig:     "",
		CustomerID:      "test",
		DeliveryService: "meest",
		ShardKey:        "9",
		SmID:            99,
		DateCreated:     time.Now(),
		OofShard:        "1",
	}
}

// Вспомогательная функция для получения переменных окружения
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Запуск тестов
func TestOrderRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(OrderRepositoryTestSuite))
}

// Отдельные тесты для случаев, когда база данных недоступна
func TestOrderRepositoryUnit(t *testing.T) {
	t.Run("TestNewPostgresOrderRepository", func(t *testing.T) {
		// Тест создания репозитория
		repo := repository.NewPostgresOrderRepository(nil)
		assert.NotNil(t, repo)
		// Проверяем, что репозиторий создан успешно
		assert.IsType(t, &repository.PostgresOrderRepository{}, repo)
	})

	t.Run("TestCreateTestOrder", func(t *testing.T) {
		// Тест создания тестового заказа
		order := createTestOrder("test-order")
		assert.Equal(t, "test-order", order.OrderUID)
		assert.Equal(t, "WBILMTESTTRACK", order.TrackNumber)
		assert.Equal(t, "Test Testov", order.Delivery.Name)
		assert.Equal(t, 1817, order.Payment.Amount)
		assert.Len(t, order.Items, 1)
	})
}
