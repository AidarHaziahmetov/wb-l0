package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"wb-l0-go/internal/service"
)

type Consumer struct {
	reader *kafka.Reader
	svc    *service.OrderService
	log    *zap.Logger
}

func NewConsumer(brokers []string, topic, groupID string, svc *service.OrderService, log *zap.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.LastOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     500 * time.Millisecond,
	})
	return &Consumer{reader: reader, svc: svc, log: log}
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			c.log.Info("context done, stopping consumer")
			return ctx.Err()
		default:
			m, err := c.reader.FetchMessage(ctx)
			if err != nil {
				return err
			}
			if err := c.svc.HandleKafkaOrder(ctx, string(m.Key), m.Value); err != nil {
				c.log.Error("failed to handle message", zap.Error(err))
				// не коммитил сообщение, для повторной попытки обработки
				// continue
			}
			if err := c.reader.CommitMessages(ctx, m); err != nil {
				c.log.Error("failed to commit message", zap.Error(err))
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
