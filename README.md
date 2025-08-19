# WB L0 Go - Микросервис для обработки заказов

## Описание проекта

**WB L0 Go** микросервис на языке Go, для обработки и управления заказами

## Быстрый старт

### Предварительные требования

- Go 1.25.0 или выше
- Docker и Docker Compose
- Make (опционально)

### Запуск с Docker Compose

1. **Клонируйте репозиторий:**
```bash
git clone <repository-url>
cd wb-l0-go
```

2. **Запустите все сервисы:**
```bash
docker-compose up -d
```

3. **Проверьте статус сервисов:**
```bash
docker-compose ps
```

### Запуск локально

1. **Установите зависимости:**
```bash
make deps
go mod tidy
```

2. **Настройте переменные окружения:**
```bash
cp .env.example .env
# Отредактируйте .env файл под ваши настройки
```

3. **Запустите миграции:**
```bash
make migrate-up
```

4. **Запустите приложение:**
```bash
make run-api
```

## API Endpoints

### Swagger документация
- **URL**: `http://localhost:8080/swagger/`
- **Описание**: Интерактивная документация API

### Основные endpoints

#### 1. Получить список заказов
```
GET /orders?limit=50&offset=0
```
**Параметры:**
- `limit` (опционально) - количество заказов (по умолчанию: 50)
- `offset` (опционально) - смещение (по умолчанию: 0)

#### 2. Получить заказ по UID
```
GET /orders/{order_uid}
```
**Параметры:**
- `order_uid` (обязательно) - уникальный идентификатор заказа

#### 3. Опубликовать заказ в Kafka
```
POST /publish
```
**Тело запроса:** JSON объект заказа
**Ответ:** Статус публикации

## Конфигурация

### Переменные окружения

| Переменная | Описание | Значение по умолчанию |
|------------|----------|----------------------|
| `APP_NAME` | Название приложения | `wb-l0-go` |
| `HTTP_ADDR` | HTTP адрес сервера | `:8080` |
| `KAFKA_BROKERS` | Адреса Kafka брокеров | `localhost:9092` |
| `KAFKA_TOPIC` | Топик Kafka | `orders` |
| `KAFKA_GROUP_ID` | ID группы потребителя | `wb-l0-go-consumer` |
| `PG_DSN` | DSN PostgreSQL | `postgres://postgres:postgres@localhost:5432/l0?sslmode=disable` |
| `LOG_LEVEL` | Уровень логирования | `info` |
| `CACHE_MAX_ITEMS` | Максимальное количество элементов в кэше | `100` |

### Структура конфигурации

```go
type Config struct {
    AppName       string   `envconfig:"APP_NAME" default:"wb-l0-go"`
    HTTPAddr      string   `envconfig:"HTTP_ADDR" default:":8080"`
    KafkaBrokers  []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
    KafkaTopic    string   `envconfig:"KAFKA_TOPIC" default:"orders"`
    KafkaGroupID  string   `envconfig:"KAFKA_GROUP_ID" default:"wb-l0-go-consumer"`
    PostgresDSN   string   `envconfig:"PG_DSN" default:"postgres://postgres:postgres@localhost:5432/l0?sslmode=disable"`
    LogLevel      string   `envconfig:"LOG_LEVEL" default:"info"`
    CacheMaxItems int      `envconfig:"CACHE_MAX_ITEMS" default:"100"`
}
```

## Структура проекта

```
wb-l0-go/
├── cmd/                    # Точки входа приложения
│   └── app/
│       └── main.go        # Основная функция main
├── internal/               # Внутренние пакеты
│   ├── cache/             # Слой кэширования
│   ├── config/            # Конфигурация
│   ├── db/                # Подключение к БД
│   ├── domain/            # Доменные модели
│   ├── frontend/          # Статические файлы
│   ├── logger/            # Логирование
│   ├── repository/        # Слой доступа к данным
│   ├── service/           # Бизнес-логика
│   └── transport/         # Транспортный слой
│       ├── http/          # HTTP handlers
│       └── kafka/         # Kafka producer/consumer
├── migrations/             # Миграции базы данных
├── docs/                   # Swagger документация
├── docker-compose.yml      # Docker Compose конфигурация
├── Dockerfile             # Docker образ
├── Makefile               # Команды для сборки и запуска
├── go.mod                 # Go модули
└── README.md              # Документация
```

## Команды Make

### Основные команды

```bash
# Запуск приложения
make run-app


# Сборка приложения
make build

# Управление зависимостями
make deps
make tidy

# Тестирование
make test              # Все тесты

# Миграции БД
make migrate-install   # Установка golang-migrate
make migrate-create    # Создание новой миграции
make migrate-up        # Применение миграций
make migrate-down      # Откат миграций
```

### Docker команды

```bash
# Запуск всех сервисов
docker-compose up -d

# Остановка всех сервисов
docker-compose down

# Просмотр логов
docker-compose logs -f app

# Пересборка и перезапуск
docker-compose up -d --build
```

### Создание миграций

```bash
make migrate-create name=add_new_table
```

### Запуск тестов

```bash
# Все тесты
make test

# Конкретный пакет
go test -v ./internal/service/

```

## Мониторинг и логирование

### Логирование

Приложение использует структурированное логирование с помощью Zap:

```go
log.Info("server started", zap.String("addr", cfg.HTTPAddr))
log.Error("failed to process order", zap.Error(err), zap.String("order_uid", orderUID))
```

### Kafka UI

Доступен веб-интерфейс для мониторинга Kafka:
- **URL**: `http://localhost:8081`
- **Описание**: Управление топиками, просмотр сообщений

