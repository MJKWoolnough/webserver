package main

import (
	"database/sql"
	"runtime"
	"sync"

	_ "github.com/mxk/go-sqlite/sqlite3"
)

type ConnPool struct {
	pool sync.Pool
}

type Conn struct {
	db *sql.DB
}

func (c *Conn) Close() error {
	c.db.Close()
	return nil
}

var SQLPool ConnPool

func SetupSQL(databaseName string) {
	SQLPool.pool = sync.Pool{
		New: func() interface{} {
			db, err := sql.Open("sqlite3", databaseName)
			if err != nil {
				panic(err)
			}
			// prepare statements
			c := &Conn{
				db,
			}
			runtime.SetFinalizer(c, (*Conn).Close)
			return c
		},
	}
}

func (cp *ConnPool) Connect() (c *Conn, err error) {
	defer func() {
		if e := recover(); e != nil {
			err, _ = e.(error)
		}
	}()
	return cp.pool.Get().(*Conn), nil
}
