package main

import (
	"database/sql"

	_ "github.com/mxk/go-sqlite/sqlite3"
)

type ConnPool struct {
	databaseName string
	pool         chan *Conn
}

const (
	numStatements = iota
)

type Conn struct {
	db     *sql.DB
	stmnts [numStatements]*sql.Stmt
}

func (c *Conn) Close() error {
	for _, s := range c.stmnts {
		s.Close()
	}
	return c.db.Close()
}

var SQLPool ConnPool

func SetupDatabase(databaseName string, conns uint) {
	SQLPool.databaseName = databaseName
	SQLPool.pool = make(chan *Conn, conns)
}

func (cp *ConnPool) Get() (*Conn, error) {
	select {
	case c := <-cp.pool:
		return c, nil
	default:
	}
	db, err := sql.Open("sqlite3", cp.databaseName)
	if err != nil {
		return nil, err
	}
	// prepare statements
	return &Conn{
		db: db,
	}, nil
}

func (cp *ConnPool) Put(c *Conn) {
	select {
	case cp.pool <- c:
		return
	default:
	}
	c.Close()
}
