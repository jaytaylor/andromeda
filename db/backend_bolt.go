package db

import (
	"fmt"
	"sync"
	"time"

	bolt "github.com/coreos/bbolt"
)

type BoltConfig struct {
	DBFile      string
	BoltOptions *bolt.Options
}

func NewBoltConfig(dbFilename string) *BoltConfig {
	cfg := &BoltConfig{
		DBFile: dbFilename,
		BoltOptions: &bolt.Options{
			Timeout: 1 * time.Second,
		},
	}
	return cfg
}

func (cfg BoltConfig) Type() Type {
	return Bolt
}

type BoltBackend struct {
	config *BoltConfig
	db     *bolt.DB
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
		for _, name := range tables {
			if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
				return fmt.Errorf("initDB: creating bucket %q: %s", name, err)
			}
		}
		return nil
	})
}

func (be *BoltBackend) New(name string) (Backend, error) {
	n := NewBoltBackend(NewBoltConfig(name))
	return n, nil
}

func (be *BoltBackend) Get(table string, key []byte) ([]byte, error) {
	var v []byte
	if err := be.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(table))
		if b == nil {
			return ErrKeyNotFound
		}
		v = b.Get(key)
		if len(v) == 0 {
			return ErrKeyNotFound
		}
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

func (be *BoltBackend) Destroy(tables ...string) error {
	return be.db.Update(func(tx *bolt.Tx) error {
		for _, table := range tables {
			if err := tx.DeleteBucket([]byte(table)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (be *BoltBackend) Len(table string) (int, error) {
	var n int
	if err := be.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(table))
		n = b.Stats().KeyN
		return nil
	}); err != nil {
		return 0, err
	}
	return n, nil
}

func (be *BoltBackend) Begin(writable bool) (Transaction, error) {
	tx, err := be.db.Begin(writable)
	if err != nil {
		return nil, err
	}
	bTx := be.wrapTx(tx)
	return bTx, nil
}

func (be *BoltBackend) View(fn func(tx Transaction) error) error {
	return be.db.View(func(tx *bolt.Tx) error {
		bTx := be.wrapTx(tx)
		return fn(bTx)
	})
}
func (be *BoltBackend) Update(fn func(tx Transaction) error) error {
	return be.db.Update(func(tx *bolt.Tx) error {
		bTx := be.wrapTx(tx)
		return fn(bTx)
	})
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

func (be *BoltBackend) EachTable(fn func(table string, tx Transaction) error) error {
	return be.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(table []byte, b *bolt.Bucket) error {
			bTx := be.wrapTx(b.Tx())
			if err := fn(string(table), bTx); err != nil {
				return err
			}
			return nil
		})
	})
}

func (be *BoltBackend) wrapTx(tx *bolt.Tx) *boltTx {
	bTx := &boltTx{
		tx: tx,
		be: be,
	}
	return bTx
}

type boltTx struct {
	tx *bolt.Tx
	be *BoltBackend
}

func (btx *boltTx) Get(table string, key []byte) ([]byte, error) {
	b := btx.tx.Bucket([]byte(table))
	v := b.Get(key)
	if len(v) == 0 {
		return nil, ErrKeyNotFound
	}
	return v, nil
}

func (btx *boltTx) Put(table string, key []byte, value []byte) error {
	b, err := btx.tx.CreateBucketIfNotExists([]byte(table))
	if err != nil {
		return err
	}
	return b.Put(key, value)
}

func (btx *boltTx) Delete(table string, keys ...[]byte) error {
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

func (btx *boltTx) Commit() error {
	return btx.tx.Commit()
}

func (btx *boltTx) Rollback() error {
	return btx.tx.Rollback()
}

func (btx *boltTx) Backend() Backend {
	return btx.be
}

func (btx *boltTx) Cursor(_ string) Cursor {
	c := newBoltCursor(btx.tx.Cursor())
	return c
}

type boltCursor struct {
	c *bolt.Cursor
	k []byte
	v []byte
}

func newBoltCursor(c *bolt.Cursor) Cursor {
	bc := &boltCursor{
		c: c,
	}
	return bc
}

func (bc *boltCursor) First() Cursor {
	bc.k, bc.v = bc.c.First()
	return bc
}

func (bc *boltCursor) Next() Cursor {
	bc.k, bc.v = bc.c.Next()
	return bc
}

func (bc *boltCursor) Seek(prefix []byte) Cursor {
	bc.k, bc.v = bc.c.Seek(prefix)
	return bc
}

func (bc *boltCursor) Data() (key []byte, value []byte) {
	return bc.k, bc.v
}

func (bc *boltCursor) Close() {
	bc.discard()
	return
}

func (bc *boltCursor) discard() {
	bc.k = nil
	bc.v = nil
}
func (bc *boltCursor) Err() error {
	return nil
}
