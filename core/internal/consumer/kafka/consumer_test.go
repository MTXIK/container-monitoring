package kafka

import (
	"context"
	"errors"
	"testing"

	kafkago "github.com/segmentio/kafka-go"
)

type scriptedReader struct {
	messages []kafkago.Message
	fetches  int
	commits  int
}

func (r *scriptedReader) FetchMessage(ctx context.Context) (kafkago.Message, error) {
	if r.fetches >= len(r.messages) {
		return kafkago.Message{}, context.Canceled
	}
	message := r.messages[r.fetches]
	r.fetches++
	return message, nil
}

func (r *scriptedReader) CommitMessages(ctx context.Context, messages ...kafkago.Message) error {
	r.commits += len(messages)
	return nil
}

func (r *scriptedReader) Close() error {
	return nil
}

type failingOnceHandler struct {
	calls int
}

func (h *failingOnceHandler) Handle(ctx context.Context, message Message) error {
	h.calls++
	if h.calls == 1 {
		return errors.New("temporary handler error")
	}
	return nil
}

func TestConsumerContinuesAfterHandlerError(t *testing.T) {
	reader := &scriptedReader{messages: []kafkago.Message{
		{Topic: "container.metrics", Value: []byte("bad")},
		{Topic: "container.metrics", Value: []byte("good")},
	}}
	handler := &failingOnceHandler{}
	consumer := NewConsumerWithReader(reader)

	err := consumer.Run(context.Background(), handler)

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if handler.calls != 2 {
		t.Fatalf("handler calls = %d, want 2", handler.calls)
	}
	if reader.commits != 2 {
		t.Fatalf("commits = %d, want 2", reader.commits)
	}
}

func TestConsumerInitializesTopicsBeforeCreatingReader(t *testing.T) {
	script := &scriptedReader{}
	calls := []string{}
	consumer := NewConsumerWithReader(script)
	consumer.topicInitializer = func(context.Context) error {
		calls = append(calls, "init")
		return nil
	}
	consumer.readerFactory = func() reader {
		calls = append(calls, "reader")
		return script
	}

	err := consumer.Run(context.Background(), &failingOnceHandler{})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := calls, []string{"init", "reader"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("calls = %v, want %v", got, want)
	}
}
