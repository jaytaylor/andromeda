package db

type Type int

const (
	Bolt Type = iota
	Postgres
	Rocks
)
