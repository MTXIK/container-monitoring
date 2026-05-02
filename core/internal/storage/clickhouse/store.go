package clickhouse

import "context"

type Store struct {
	dsn string
}

func New(dsn string) *Store {
	return &Store{dsn: dsn}
}

func (s *Store) Ping(ctx context.Context) error {
	_ = ctx
	_ = s.dsn
	return nil
}
