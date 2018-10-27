package db

type Queue interface {
	Open() error
	Close() error
	Enqueue(topic string, priority int, values ...[]byte) error
	Dequeue(topic string) (value []byte, err error)
	Scan(name string, opts *QueueOptions, fn func(value []byte)) error
	ScanWithBreak(name string, opts *QueueOptions, fn func(value []byte) bool) error
	Len(name string, priority int) (int, error)
	Destroy(topics ...string) error
}
