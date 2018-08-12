package db

// Backend is a generic K/V persistence interface.
type Backend interface {
	Open() error
	Close() error
	New(name string) (Backend, error) // Returns a new instance of the same backend type with a different data file.
	Get(table string, key []byte) (value []byte, err error)
	Put(table string, key []byte, value []byte) error
	Delete(table string, keys ...[]byte) error
	Drop(tables ...string) error
	Len(table string) (n int, err error)
	Begin(writable bool) (Transaction, error)
	View(fn func(tx Transaction) error) error // Read-only.
	Update(fn func(tx Transaction) error) error
	EachRow(table string, fn func(key []byte, value []byte)) error               // Read-only.
	EachRowWithBreak(table string, fn func(key []byte, value []byte) bool) error // Read-only.
	EachTable(func(table string, tx Transaction) error) error                    // Read-only.
}

// Transaction is a generic Tx interface to be provided by each Backend
// implementation.
type Transaction interface {
	Get(table string, key []byte) (value []byte, err error)
	Put(table string, key []byte, value []byte) error
	Delete(table string, keys ...[]byte) error
	Commit() error
	Rollback() error
	Cursor(table string) Cursor
	Backend() Backend
}

// Cursor is a generic interface to be provided by each Backend implementation.
// Expects an object-builder style implementation to allow easy access to .Data.
type Cursor interface {
	// First moves the cursor to the beginning of the range of elements.
	First() Cursor

	// Last moves the cursor to the end of the range of elements.
	Last() Cursor

	// Next moves the cursor to the next element.
	Next() Cursor

	// Prev moves the cursor to the previous element.
	Prev() Cursor

	// Seek moves the
	Seek(prefix []byte) Cursor

	// Data returns the K/V pair at the current cursor position.
	// Returns (nil, nil) when past the end.
	Data() (key []byte, value []byte)

	// Close cleans up and returns resources to the system.
	Close()
}
