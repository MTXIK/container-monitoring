package redis

import "context"

type Store struct {
	addr string
}

func New(addr string) *Store {
	return &Store{addr: addr}
}

func (s *Store) Ping(ctx context.Context) error {
	_ = ctx
	_ = s.addr
	return nil
}
