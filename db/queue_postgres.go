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

	if err := q.initDB(); err != nil {
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

func (q *PostgresQueue) initDB() error {
	for _, table := range qTables {
		table = q.normalizeTable(table)
		_, err := q.db.Exec(fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %v (
	id SERIAL PRIMARY KEY,
	priority SMALLINT NOT NULL,
	value BYTEA NOT NULL
)
`, pq.QuoteIdentifier(table)))
		if err != nil {
			return fmt.Errorf("creating table %q: %s", table, err)
		}
		_, err = q.db.Exec(fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %v ON %v (%v)`,
			pq.QuoteIdentifier(fmt.Sprintf("%v_priority_index", table)),
			pq.QuoteIdentifier(table),
			pq.QuoteIdentifier("priority"),
		))
		if err != nil {
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
		priority = numPriorities
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
	tx, err := q.db.Begin()
	if err != nil {
		return nil, err
	}

	bestEffortRB := func() error {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.WithField("table", table).Errorf("Rolling back due to error deleting row: %s", rbErr)
			return rbErr
		}
		return nil
	}

	row := tx.QueryRow(fmt.Sprintf(
		`SELECT id, priority, value FROM %v ORDER BY priority, id ASC LIMIT 1`,
		pq.QuoteIdentifier(q.normalizeTable(table)),
	))

	var (
		id int64
		p  int
		v  []byte
	)
	if err = row.Scan(&id, &p, &v); err != nil {
		if err == sql.ErrNoRows {
			if err = bestEffortRB(); err != nil {
				return nil, err
			}
			return nil, io.EOF
		}
		return nil, fmt.Errorf("scanning row values: %s", err)
	}

	_, err = tx.Exec(
		fmt.Sprintf(`DELETE FROM %v WHERE id=$1`, pq.QuoteIdentifier(q.normalizeTable(table))),
		id,
	)
	if err != nil {
		bestEffortRB()
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return v, nil
}

func (q *PostgresQueue) Scan(table string, opts *QueueOptions, fn func(value []byte)) error {
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
		fn(value)
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
