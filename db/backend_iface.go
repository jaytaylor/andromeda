package db

// Backend is a generic K/V persistence interface.
type Backend interface {
	Open() error
	Close() error
	Get(table string, key []byte) (value []byte, err error)
	Put(table string, key []byte, value []byte) error
	Delete(table string, keys ...[]byte) error
	Drop(tables ...string) error
	Len(table string) (n int, err error)
	WithTransaction(opts TXOptions, fn func(tx Transaction) error) error
	EachRow(table string, fn func(key []byte, value []byte)) error
	EachRowWithBreak(table string, fn func(key []byte, value []byte) bool) error
	Enqueue(table string, priority int, value []byte) error
	Dequeue(table string) (value []byte, err error)
	Cursor() Cursor
}

// Transaction is a generic TX interface to be provided by each Backend
// implementation.
type Transaction interface {
	Get(table string, key []byte) (value []byte, err error)
	Put(table string, key []byte, value []byte) error
	Delete(table string, keys ...[]byte) error
	Commit() error
	Rollback() error
	Cursor() Cursor
}

type TXOptions struct {
	ReadOnly bool
}

// Cursor is a generic interface to be provided by each Backend implementation.
type Cursor interface {
	// First moves the cursor to the beginning of the range of elements.
	First()

	// Last moves the cursor to the end of the range of elements.
	Last()

	// Next moves the cursor to the next element.
	Next() (key []byte, value []byte)

	// Prev moves the cursor to the previous element.
	Prev() (key []byte, value []byte)

	// Seek moves the
	Seek(prefix []byte) (key []byte, value []byte)

	// Data returns the K/V pair at the current cursor position.
	// Returns (nil, nil) when past the end.
	Data() (key []byte, value []byte)

	// Close cleans up and returns resources to the system.
	Close()
}
