package kafka

import (
	"context"

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
	brokers []string
	topics  []string
	groupID string
}

func NewConsumer(brokers []string, topics []string, groupID string) *Consumer {
	return &Consumer{brokers: brokers, topics: topics, groupID: groupID}
}

func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     c.brokers,
		GroupID:     c.groupID,
		Topic:       "",
		GroupTopics: c.topics,
	})
	defer reader.Close()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if err := handler.Handle(ctx, Message{
			Topic: msg.Topic,
			Key:   msg.Key,
			Value: msg.Value,
		}); err != nil {
			return err
		}
		if err := reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}
