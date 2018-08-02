package db

type Backend interface {
	Open() error
	Close() error
	Get(table string, key []byte) (value []byte, err error)
	Put(table string, key []byte, value []byte) error
	Len(table string) (n int, err error)
	WithTransaction(opts TXOptions, fn func(tx Transaction) error) error
	EachRow(table string, fn func(key []byte, value []byte)) error
	EachRowWithBreak(table string, fn func(key []byte, value []byte) bool) error
	Enqueue(table string, priority int, value []byte) error
	Dequeue(table string) (value []byte, err error)
}

type Transaction interface {
	Get(table string, key []byte) (value []byte, err error)
	Put(table string, key []byte, value []byte) error
	Commit() error
	Rollback() error
}

type TXOptions struct {
	ReadOnly bool
}
