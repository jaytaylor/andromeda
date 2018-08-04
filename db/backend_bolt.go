package db

import (
	"fmt"
	"sync"

	bolt "github.com/coreos/bbolt"
	boltqueue "jaytaylor.com/bboltqueue"
)

type BoltBackend struct {
	config *BoltConfig
	db     *bolt.DB
	q      *boltqueue.PQueue
	mu     sync.Mutex
}

func NewBoltBackend(config *BoltConfig) *BoltBackend {
	be := &BoltBackend{
		config: config,
	}
	return be
}

func (be *BoltBackend) Open() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.db != nil {
		return nil
	}

	db, err := bolt.Open(be.config.DBFile, 0600, be.config.BoltOptions)
	if err != nil {
		return err
	}
	be.db = db

	be.q = boltqueue.NewPQueue(be.db)

	if err := be.initDB(); err != nil {
		return err
	}

	return nil
}

func (be *BoltBackend) Close() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.db == nil {
		return nil
	}

	if err := be.db.Close(); err != nil {
		return err
	}

	be.db = nil

	return nil
}

func (be *BoltBackend) initDB() error {
	return be.db.Update(func(tx *bolt.Tx) error {
		buckets := []string{
			TableMetadata,
			TablePackages,
			TableToCrawl,
			TablePendingReferences,
		}
		for _, name := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
				return fmt.Errorf("initDB: creating bucket %q: %s", name, err)
			}
		}
		return nil
	})
}

func (be *BoltBackend) Get(table string, key []byte) ([]byte, error) {
	var v []byte
	if err := be.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(table))
		if b == nil {
			return ErrKeyNotFound
		}
		v = b.Get(key)
		return nil
	}); err != nil {
		return nil, err
	}
	return v, nil
}

func (be *BoltBackend) Put(table string, key []byte, value []byte) error {
	return be.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(table))
		if err != nil {
			return err
		}
		return b.Put(key, value)
	})
}

func (be *BoltBackend) Delete(table string, keys ...[]byte) error {
	return be.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(table))
		if err != nil {
			return err
		}
		for _, key := range keys {
			if err := b.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

func (be *BoltBackend) Drop(tables ...string) error {
	return be.db.Update(func(tx *bolt.Tx) error {
		for _, table := range tables {
			if err := tx.DeleteBucket([]byte(table)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (be *BoltBackend) WithTransaction(opts TXOptions, fn func(tx Transaction) error) error {
	if opts.ReadOnly {
		return be.db.View(func(tx *bolt.Tx) error {
			btx := be.wrapTX(tx)
			return fn(btx)
		})
	}
	if err := be.db.Update(func(tx *bolt.Tx) error {
		btx := be.wrapTX(tx)
		return fn(btx)
	}); err != nil {
		return err
	}
	return nil
}

func (be *BoltBackend) EachRow(table string, fn func(key []byte, value []byte)) error {
	return be.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(table))
			c = b.Cursor()
		)
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fn(k, v)
		}
		return nil
	})
}

func (be *BoltBackend) EachRowWithBreak(table string, fn func(key []byte, value []byte) bool) error {
	return be.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(table))
			c = b.Cursor()
		)
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if !fn(k, v) {
				break
			}
		}
		return nil
	})
}

func (be *BoltBackend) Len(table string) (int, error) {
	var n int
	if err := be.db.View(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(table))
		if err != nil {
			return err
		}
		n = b.Stats().KeyN
		return nil
	}); err != nil {
		return 0, err
	}
	return n, nil
}

func (be *BoltBackend) Enqueue(table string, priority int, value []byte) error {
	m := boltqueue.NewMessage(string(value))
	if err := be.q.Enqueue(table, priority, m); err != nil {
		return err
	}
	return nil
}

func (be *BoltBackend) Dequeue(table string) ([]byte, error) {
	v, err := be.q.Dequeue(table)
	if err != nil {
		return nil, err
	}
	return v.Value, nil
}

func (be *BoltBackend) wrapTX(tx *bolt.Tx) *boltTX {
	bTX := &boltTX{
		tx: tx,
	}
	return bTX
}

type boltTX struct {
	tx *bolt.Tx
}

func (btx *boltTX) Get(table string, key []byte) ([]byte, error) {
	b, err := btx.tx.CreateBucketIfNotExists([]byte(table))
	if err != nil {
		return nil, err
	}
	v := b.Get(key)
	return v, nil
}

func (btx *boltTX) Put(table string, key []byte, value []byte) error {
	b, err := btx.tx.CreateBucketIfNotExists([]byte(table))
	if err != nil {
		return err
	}
	return b.Put(key, value)
}

func (btx *boltTX) Delete(table string, keys ...[]byte) error {
	b, err := btx.tx.CreateBucketIfNotExists([]byte(table))
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := b.Delete(key); err != nil {
			return err
		}
	}
	return nil
}

func (btx *boltTX) Commit() error {
	return btx.tx.Commit()
}

func (btx *boltTX) Rollback() error {
	return btx.tx.Rollback()
}
