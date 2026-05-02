package kafka

import "context"

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
}

func NewConsumer(brokers []string, topics []string) *Consumer {
	return &Consumer{brokers: brokers, topics: topics}
}

func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	_ = c.brokers
	_ = c.topics
	_ = handler
	<-ctx.Done()
	return nil
}
