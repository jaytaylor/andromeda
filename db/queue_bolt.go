package db

import (
	"bytes"
	"sync"

	bolt "github.com/coreos/bbolt"
	boltqueue "jaytaylor.com/bboltqueue"
)

type BoltQueue struct {
	db *bolt.DB
	q  *boltqueue.PQueue
	mu sync.Mutex
}

func NewBoltQueue(db *bolt.DB) *BoltQueue {
	bq := &BoltQueue{
		db: db,
	}
	return bq
}

func (bq *BoltQueue) Open() error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	if bq.q != nil {
		return nil
	}

	bq.q = boltqueue.NewPQueue(bq.db)

	return nil
}

func (bq *BoltQueue) Close() error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	if bq.q == nil {
		return nil
	}

	if err := bq.q.Close(); err != nil {
		return err
	}
	bq.q = nil

	return nil
}

func (bq *BoltQueue) Enqueue(table string, priority int, values ...[]byte) error {
	msgs := make([]*boltqueue.Message, 0, len(values))
	for _, value := range values {
		m := boltqueue.NewMessage(string(value))
		msgs = append(msgs, m)
	}
	if err := bq.q.Enqueue(table, priority, msgs...); err != nil {
		return err
	}
	return nil
}

func (bq *BoltQueue) Dequeue(table string) ([]byte, error) {
	v, err := bq.q.Dequeue(table)
	if err != nil {
		return nil, err
	}
	return v.Value, nil
}

func (bq *BoltQueue) Scan(name string, opts *QueueOptions, fn func(value []byte)) error {
	return bq.q.Scan(name, func(m *boltqueue.Message) {
		if opts != nil {
			if p := m.Priority(); p > 0 && p != opts.Priority {
				return
			}
		}
		fn(m.Value)
	})
}

func (bq *BoltQueue) ScanWithBreak(name string, opts *QueueOptions, fn func(value []byte) bool) error {
	return bq.q.ScanWithBreak(name, func(m *boltqueue.Message) bool {
		if opts != nil {
			if p := m.Priority(); p > 0 && p != opts.Priority {
				return true
			}
		}
		return fn(m.Value)
	})
}

func (bq *BoltQueue) Len(name string, priority int) (int, error) {
	if priority <= 0 {
		return bq.q.Len(name, priority)
	}
	sum := 0
	for i := 1; i <= MaxPriority; i++ {
		n, err := bq.q.Len(name, i)
		if err != nil {
			return 0, err
		}
		sum += n
	}
	return sum, nil
}

// Destroy completely eliminates the named queue topics.
func (bq *BoltQueue) Destroy(topics ...string) error {
	return bq.db.View(func(tx *bolt.Tx) error {
		for _, topic := range topics {
			err := tx.ForEach(func(bucket []byte, _ *bolt.Bucket) error {
				if len(topic) > 0 && bytes.HasPrefix(bucket, []byte(topic)) {
					return tx.DeleteBucket(bucket)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
