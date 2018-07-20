package feed

import (
	"encoding/json"
	"fmt"
	"time"

	"jaytaylor.com/andromeda/db"
)

type timestamped struct {
	c    db.Client
	name string
}

func newTimestamped(dbClient db.Client, name string) *timestamped {
	stamped := &timestamped{
		c:    dbClient,
		name: name,
	}
	return stamped
}

func (stamped *timestamped) last() (*time.Time, error) {
	var (
		data = []byte{}
		ts   = &time.Time{}
	)
	if err := stamped.c.Meta(stamped.key(), &data); err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(data, ts); err != nil {
		return nil, err
	}
	return ts, nil
}

func (stamped *timestamped) mark(ts time.Time) error {
	data, err := json.Marshal(&ts)
	if err != nil {
		return err
	}
	if err = stamped.c.MetaSave(stamped.key(), data); err != nil {
		return err
	}
	return nil
}

func (stamped timestamped) key() string {
	return fmt.Sprintf("timestamped_%v", stamped.name)
}
