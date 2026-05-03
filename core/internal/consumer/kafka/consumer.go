package kafka

import (
	"context"
	"errors"
	"log/slog"
	"os"

	kafkago "github.com/segmentio/kafka-go"
)

type Message struct {
	Topic string
	Key   []byte
	Value []byte
}

type Handler interface {
	Handle(ctx context.Context, message Message) error
}

type Consumer struct {
	brokers       []string
	topics        []string
	groupID       string
	readerFactory func() reader
	logger        *slog.Logger
}

func NewConsumer(brokers []string, topics []string, groupID string) *Consumer {
	consumer := &Consumer{brokers: brokers, topics: topics, groupID: groupID, logger: slog.New(slog.NewTextHandler(os.Stdout, nil))}
	consumer.readerFactory = func() reader {
		return kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:     consumer.brokers,
			GroupID:     consumer.groupID,
			Topic:       "",
			GroupTopics: consumer.topics,
		})
	}
	return consumer
}

func NewConsumerWithReader(r reader) *Consumer {
	return &Consumer{
		readerFactory: func() reader { return r },
		logger:        slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	reader := c.readerFactory()
	defer reader.Close()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if err := handler.Handle(ctx, Message{
			Topic: msg.Topic,
			Key:   msg.Key,
			Value: msg.Value,
		}); err != nil {
			c.logger.Error("kafka message handler failed", "topic", msg.Topic, "error", err)
		}
		if err := reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("kafka commit failed", "topic", msg.Topic, "error", err)
		}
	}
}

type reader interface {
	FetchMessage(ctx context.Context) (kafkago.Message, error)
	CommitMessages(ctx context.Context, messages ...kafkago.Message) error
	Close() error
}
