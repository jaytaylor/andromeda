// +build rocks

package db

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"

	"gigawatt.io/errorlib"

	log "github.com/sirupsen/logrus"
	rocks "github.com/tecbot/gorocksdb"
)

type RocksConfig struct {
	Dir            string
	RocksOptions   *rocks.Options
	RocksTXOptions *rocks.TransactionDBOptions
}

func NewRocksConfig(dir string) *RocksConfig {
	cfg := &RocksConfig{
		Dir:            dir,
		RocksOptions:   rocks.NewDefaultOptions(),
		RocksTXOptions: rocks.NewDefaultTransactionDBOptions(),
	}
	cfg.RocksOptions.SetCreateIfMissing(true)
	cfg.RocksOptions.SetCreateIfMissingColumnFamilies(true)
	return cfg
}

func (cfg RocksConfig) Type() Type {
	return Rocks
}

type RocksBackend struct {
	config *RocksConfig
	dbs    map[string]*rocks.TransactionDB
	mu     sync.Mutex
}

func NewRocksBackend(config *RocksConfig) *RocksBackend {
	be := &RocksBackend{
		config: config,
	}
	return be
}

func (be *RocksBackend) Open() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.dbs != nil {
		return nil
	}
	be.dbs = map[string]*rocks.TransactionDB{}

	for _, table := range []string{
		TablePackages,
		TablePendingReferences,
		TableMetadata,
	} {
		db, err := be.db(table, true)
		if err != nil {
			return err
		}
		be.dbs[table] = db
	}

	return nil
}

func (be *RocksBackend) Close() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.dbs == nil {
		return nil
	}

	for _, db := range be.dbs {
		db.Close()
	}

	be.dbs = nil

	return nil
}

func (be *RocksBackend) db(table string, skipLocking ...bool) (*rocks.TransactionDB, error) {
	if len(skipLocking) == 0 || !skipLocking[0] {
		be.mu.Lock()
		defer be.mu.Unlock()
	}

	if db, ok := be.dbs[table]; ok {
		return db, nil
	}
	dbPath := filepath.Join(be.config.Dir, table)
	db, err := rocks.OpenTransactionDb(be.config.RocksOptions, be.config.RocksTXOptions, dbPath)
	if err != nil {
		return nil, err
	}
	be.dbs[table] = db
	return db, nil
}

func (be *RocksBackend) New(dir string) (Backend, error) {
	if dir == be.config.Dir {
		return nil, errors.New("new dir must be different from existing value")
	}
	cfg := &RocksConfig{
		Dir:            dir,
		RocksOptions:   rocks.NewDefaultOptions(),
		RocksTXOptions: rocks.NewDefaultTransactionDBOptions(),
	}

	n := NewRocksBackend(cfg)
	return n, nil
}

func (be *RocksBackend) Get(table string, key []byte) ([]byte, error) {
	db, err := be.db(table)
	if err != nil {
		return nil, err
	}
	sl, err := db.Get(rocks.NewDefaultReadOptions(), key)
	if err != nil {
		return nil, err
	}
	defer sl.Free()
	if sl.Size() == 0 {
		return nil, ErrKeyNotFound
	}
	v := make([]byte, sl.Size())
	copy(v, sl.Data())
	return v, nil
}

func (be *RocksBackend) Put(table string, key []byte, value []byte) error {
	db, err := be.db(table)
	if err != nil {
		return err
	}
	if err := db.Put(rocks.NewDefaultWriteOptions(), key, value); err != nil {
		return err
	}
	return nil
}

func (be *RocksBackend) Delete(table string, keys ...[]byte) error {
	tx := be.newTx()
	for _, key := range keys {
		if err := tx.Delete(table, key); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Errorf("Rollback failed: %s (pre-existing error=%s)", rbErr, err)
			}
			return err
		}
	}
	return tx.Commit()
}

func (be *RocksBackend) Drop(tables ...string) error {
	if err := be.Close(); err != nil {
		return err
	}

	errs := []error{}
	for _, table := range tables {
		if err := rocks.DestroyDb(table, be.config.RocksOptions); err != nil {
			errs = append(errs, fmt.Errorf("purging %q: %s", table, err))
		}
	}
	if err := errorlib.Merge(errs); err != nil {
		return err
	}

	return be.Open()
}

// Len is only able to provide a best estimate because RocksDB does not track
// exact counts.
func (be *RocksBackend) Len(table string) (int, error) {
	opts := rocks.NewDefaultOptions()
	dbPath := filepath.Join(be.config.Dir, table)
	db, err := rocks.OpenDbForReadOnly(opts, dbPath, false)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	prop := db.GetProperty("rocksdb.estimate-num-keys")
	n, err := strconv.Atoi(prop)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Begin method in the RocksDB Backend implementation is not able to support the
// "writable" flag.
func (be *RocksBackend) Begin(_ bool) (Transaction, error) {
	rTx := be.newTx()
	return rTx, nil
}

// View is read / write capable, just an alias to the update() method.
func (be *RocksBackend) View(fn func(tx Transaction) error) error {
	rTx := be.newTx()
	defer func() {
		if rbErr := rTx.Rollback(); rbErr != nil {
			log.Errorf("Unexpected error rolling back after .View: %s", rbErr)
		}
	}()
	if err := fn(rTx); err != nil {
		return err
	}
	return nil
}
func (be *RocksBackend) Update(fn func(tx Transaction) error) error {
	rTx := be.newTx()
	if err := fn(rTx); err != nil {
		if rbErr := rTx.Rollback(); rbErr != nil {
			log.Errorf("Unexpected error rolling back: %s (existing err=%v)", rbErr, err)
		}
		return err
	}
	return rTx.Commit()
}

func (be *RocksBackend) EachRow(table string, fn func(key []byte, value []byte)) error {
	rTx := be.newTx()
	c := rTx.Cursor(table).First()
	defer c.Close()
	for {
		k, v := c.Data()
		if len(k) == 0 {
			break
		}
		fn(k, v)
		c.Next()
	}
	return rTx.Rollback()
}

func (be *RocksBackend) EachRowWithBreak(table string, fn func(key []byte, value []byte) bool) error {
	rTx := be.newTx()
	c := rTx.Cursor(table).First()
	defer c.Close()
	for {
		k, v := c.Data()
		if len(k) == 0 {
			break
		}
		if !fn(k, v) {
			break
		}
		c.Next()
	}
	return rTx.Rollback()
}

func (be *RocksBackend) EachTable(fn func(table string, tx Transaction) error) error {
	rTx := be.newTx()
	for table, _ := range be.dbs {
		if err := fn(table, rTx); err != nil {
			return err
		}
	}
	return rTx.Rollback()
}

func (be *RocksBackend) newTx() *rocksTx {
	bTx := &rocksTx{
		txs: map[string]*rocks.Transaction{},
		be:  be,
	}
	return bTx
}

type rocksTx struct {
	txs map[string]*rocks.Transaction
	be  *RocksBackend
	mu  sync.Mutex
}

func (rtx *rocksTx) Get(table string, key []byte) ([]byte, error) {
	tx := rtx.getOrOpenTx(table)
	sl, err := tx.Get(rocks.NewDefaultReadOptions(), key)
	if err != nil {
		return nil, err
	}
	defer sl.Free()
	if sl.Size() == 0 {
		return nil, ErrKeyNotFound
	}
	v := make([]byte, sl.Size())
	copy(v, sl.Data())
	return v, nil
}

func (rtx *rocksTx) Put(table string, key []byte, value []byte) error {
	tx := rtx.getOrOpenTx(table)
	return tx.Put(key, value)
}

func (rtx *rocksTx) Delete(table string, keys ...[]byte) error {
	tx := rtx.getOrOpenTx(table)
	for _, key := range keys {
		if err := tx.Delete(key); err != nil {
			return err
		}
	}
	return nil
}

func (rtx *rocksTx) Commit() error {
	rtx.mu.Lock()
	defer rtx.mu.Unlock()

	errs := []error{}
	for _, tx := range rtx.txs {
		if err := tx.Commit(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errorlib.Merge(errs); err != nil {
		return err
	}
	return nil
}

func (rtx *rocksTx) Rollback() error {
	rtx.mu.Lock()
	defer rtx.mu.Unlock()

	errs := []error{}
	for _, tx := range rtx.txs {
		if err := tx.Rollback(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errorlib.Merge(errs); err != nil {
		return err
	}
	return nil
}

func (rtx *rocksTx) Backend() Backend {
	return rtx.be
}

func (rtx *rocksTx) Cursor(table string) Cursor {
	tx := rtx.getOrOpenTx(table)
	iter := tx.NewIterator(rocks.NewDefaultReadOptions())
	c := newRocksCursor(iter)
	return c
}

func (rtx *rocksTx) getOrOpenTx(table string) *rocks.Transaction {
	rtx.mu.Lock()
	defer rtx.mu.Unlock()

	tx, ok := rtx.txs[table]
	if !ok {
		// TODO: Create db table if not exists.
		tx = rtx.be.dbs[table].TransactionBegin(rocks.NewDefaultWriteOptions(), rocks.NewDefaultTransactionOptions(), nil)
		rtx.txs[table] = tx
	}
	return tx
}

type rocksCursor struct {
	iter *rocks.Iterator
	k    []byte
	v    []byte
}

func newRocksCursor(iter *rocks.Iterator) Cursor {
	rc := &rocksCursor{
		iter: iter,
	}
	return rc
}

func (rc *rocksCursor) First() Cursor {
	rc.iter.SeekToFirst()
	rc.copyCurrent()
	return rc
}

func (rc *rocksCursor) Next() Cursor {
	rc.iter.Next()
	rc.copyCurrent()
	return rc
}

func (rc *rocksCursor) Seek(prefix []byte) Cursor {
	rc.iter.SeekToFirst()
	for rc.iter.Valid() {
		if bytes.HasPrefix(rc.iter.Key().Data(), prefix) {
			rc.copyCurrent()
			break
		}
		rc.freeCurrent()
		rc.Next()
	}
	return rc
}

func (rc *rocksCursor) Data() (key []byte, value []byte) {
	return rc.k, rc.v
}

func (rc *rocksCursor) Close() {
	rc.iter.Close()
}

// copyCurrent copies the values from the current iterator into the
// cursor, then frees the iterator allocations.
func (rc *rocksCursor) copyCurrent() {
	if rc.iter.Valid() {
		rc.k = make([]byte, rc.iter.Key().Size())
		copy(rc.k, rc.iter.Key().Data())

		rc.v = make([]byte, rc.iter.Value().Size())
		copy(rc.v, rc.iter.Value().Data())

		rc.iter.Key().Free()
		rc.iter.Value().Free()
	} else {
		rc.k = nil
		rc.v = nil
	}
}

// freeCurrent frees the current iterator values without copying them
// in to the local cursor data.
func (rc *rocksCursor) freeCurrent() {
	rc.iter.Key().Free()
	rc.iter.Value().Free()
}

func (rc *rocksCursor) Err() error {
	return nil
}
