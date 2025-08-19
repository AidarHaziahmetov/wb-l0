# syntax=docker/dockerfile:1.6

ARG GO_VERSION=1.25

FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src

# Установим необходимые пакеты
RUN apk add --no-cache ca-certificates tzdata build-base

# Сначала скачиваем зависимости, чтобы лучше кэшировалось
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Копируем исходники
COPY . .


# Собираем статически линкованный бинарник
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/app ./cmd/app

FROM alpine:3.20 AS runtime
WORKDIR /app

# Минимально необходимое окружение
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/app /app/app
# Копируем фронтенд файлы
COPY --from=builder /src/internal/frontend /app/internal/frontend

# Для api
EXPOSE 8080
ENTRYPOINT ["/app/app"]
