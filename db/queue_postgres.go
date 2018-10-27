package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"sync"

	pq "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type PostgresQueue struct {
	config *PostgresConfig
	db     *sql.DB
	mu     sync.Mutex
}

func NewPostgresQueue(config *PostgresConfig) *PostgresQueue {
	q := &PostgresQueue{
		config: config,
	}
	return q
}

func (q *PostgresQueue) Open() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.db != nil {
		return nil
	}

	db, err := sql.Open("postgres", q.config.ConnString)
	if err != nil {
		return err
	}
	q.db = db

	db.SetMaxOpenConns(q.config.MaxConns)

	tx, err := q.db.Begin()
	if err != nil {
		return err
	}

	if err := q.initDB(tx, qTables...); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (q *PostgresQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.db == nil {
		return nil
	}

	if err := q.db.Close(); err != nil {
		return err
	}

	return nil
}

func (q *PostgresQueue) initDB(tx *sql.Tx, tables ...string) error {
	for _, table := range tables {
		table = q.normalizeTable(table)
		_, err := tx.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %v (
    id SERIAL PRIMARY KEY,
    priority SMALLINT NOT NULL,
    value BYTEA NOT NULL
)`, pq.QuoteIdentifier(table)))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("creating table %q: %s", table, err)
		}
		_, err = tx.Exec(fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %v ON %v (%v)`,
			pq.QuoteIdentifier(fmt.Sprintf("%v_priority_index", table)),
			pq.QuoteIdentifier(table),
			pq.QuoteIdentifier("priority"),
		))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("creating table %q: %s", table, err)
		}
	}
	return nil
}

func (q *PostgresQueue) Enqueue(table string, priority int, values ...[]byte) error {
	if len(values) == 0 {
		return nil
	}

	if priority <= 0 {
		// Default to lowest priority when an unsupported value is received.
		priority = MaxPriority
	}

	var (
		inserts = &bytes.Buffer{}
		args    = make([]interface{}, len(values)+1)
	)

	args[0] = priority

	for i, value := range values {
		if inserts.Len() > 0 {
			inserts.WriteByte(',')
		}
		inserts.WriteString(fmt.Sprintf("($1, $%v::BYTEA)", i+2))
		args[i+1] = value
	}

	_, err := q.db.Exec(
		fmt.Sprintf(
			`INSERT INTO %v (priority, value) VALUES %v`,
			pq.QuoteIdentifier(q.normalizeTable(table)),
			inserts.String(),
		),
		args...,
	)
	if err != nil {
		return err
	}
	return nil
}

func (q *PostgresQueue) Dequeue(table string) ([]byte, error) {
	row := q.db.QueryRow(fmt.Sprintf(
		`DELETE FROM %[1]v WHERE id = (
   SELECT id FROM %[1]v ORDER BY priority, id ASC FOR UPDATE SKIP LOCKED LIMIT 1
)
RETURNING id, priority, value`,
		pq.QuoteIdentifier(q.normalizeTable(table)),
	))

	var (
		id int64
		p  int
		v  []byte
	)
	if err := row.Scan(&id, &p, &v); err != nil {
		if err == sql.ErrNoRows {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("scanning row values: %s", err)
	}
	return v, nil
}

func (q *PostgresQueue) Scan(table string, opts *QueueOptions, fn func(value []byte)) error {
	return q.ScanWithBreak(table, opts, func(value []byte) bool {
		fn(value)
		return true
	})
}

func (q *PostgresQueue) ScanWithBreak(table string, opts *QueueOptions, fn func(value []byte) bool) error {
	var (
		query = fmt.Sprintf(`SELECT value FROM %v `, pq.QuoteIdentifier(q.normalizeTable(table)))
		args  []interface{}
	)

	if opts == nil || opts.Priority <= 0 {
		query += fmt.Sprintf(`ORDER BY id, priority ASC`)
	} else {
		query += fmt.Sprintf(`WHERE priority=$1 ORDER BY id ASC`)
		args = []interface{}{opts.Priority}
	}

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var value []byte
		if err = rows.Scan(&value); err != nil {
			return err
		}
		if !fn(value) {
			break
		}
	}
	return nil
}

func (q *PostgresQueue) Len(table string, priority int) (int, error) {
	var (
		query = fmt.Sprintf(`SELECT COUNT(*) FROM %v`, pq.QuoteIdentifier(q.normalizeTable(table)))
		args  []interface{}
	)

	if priority > 0 {
		query += fmt.Sprintf(` WHERE priority=$1`)
		args = []interface{}{priority}
	}

	row := q.db.QueryRow(query, args...)

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return int(count), nil
}

func (_ *PostgresQueue) normalizeTable(table string) string {
	table = strings.Replace(table, "-", "_", -1)
	return table
}

// Destroy completely eliminates the named queue topics.
func (q *PostgresQueue) Destroy(topics ...string) error {
	if len(topics) == 0 {
		return nil
	}

	tx, err := q.db.Begin()
	if err != nil {
		return err
	}

	bestEffortRB := func(table string) error {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.WithField("table", table).Errorf("Rolling back due to error dropping table: %s", rbErr)
			return rbErr
		}
		return nil
	}

	for _, topic := range topics {
		topic = q.normalizeTable(topic)
		_, err := tx.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %v`, pq.QuoteIdentifier(topic)))
		if err != nil {
			bestEffortRB(topic)
			return err
		}
	}

	if err = q.initDB(tx, topics...); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
