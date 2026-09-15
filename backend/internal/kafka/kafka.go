package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/domain"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	topic  string
	log    *slog.Logger
}

func NewProducer(brokers []string, topic string, log *slog.Logger) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.LeastBytes{},
			RequiredAcks:           kafka.RequireOne,
			Async:                  false,
			BatchTimeout:           10 * time.Millisecond,
			AllowAutoTopicCreation: true,
		},
		topic: topic,
		log:   log,
	}
}

func (p *Producer) PublishEvent(ctx context.Context, msg domain.KafkaEventMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal kafka message: %w", err)
	}
	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.EventID),
		Value: payload,
		Time:  time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("publish kafka message: %w", err)
	}
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

type Consumer struct {
	reader *kafka.Reader
	log    *slog.Logger
}

func NewConsumer(brokers []string, topic, groupID string, log *slog.Logger) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		}),
		log: log,
	}
}

func (c *Consumer) Fetch(ctx context.Context) (domain.KafkaEventMessage, kafka.Message, error) {
	var out domain.KafkaEventMessage
	m, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return out, m, err
	}
	if err := json.Unmarshal(m.Value, &out); err != nil {
		return out, m, fmt.Errorf("unmarshal kafka message: %w", err)
	}
	return out, m, nil
}

func (c *Consumer) Commit(ctx context.Context, msg kafka.Message) error {
	return c.reader.CommitMessages(ctx, msg)
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
