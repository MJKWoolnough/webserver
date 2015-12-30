package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/MJKWoolnough/webserver/proxy/client"
)

var (
	databaseFile = flag.String("d", "./database.db", "database file")
	templateDir  = flag.String("t", "./templates", "template directory")
	filesDir     = flag.String("f", "./files", "files directory")
	logName      = flag.String("n", "", "name for logging")
	logger       *log.Logger
)

func main() {
	flag.Parse()
	logger = log.New(os.Stderr, *logName, log.LstdFlags)

	db, err := sql.Open("sqlite3", *databaseFile)
	if err != nil {
		log.Printf("error while opening database: %s\n", err)
		return
	}
	defer db.Close()

	// load items from database
	// load schedule from database

	cc := make(chan struct{})
	go func() {
		logger.Println("Server Started")
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, os.Interrupt)
		select {
		case <-sc:
			logger.Println("Closing")
		case <-cc:
		}
		signal.Stop(sc)
		close(sc)
		client.Close()
		client.Wait()
		close(cc)
	}()

	err = client.Run()

	select {
	case <-cc:
	default:
		logger.Println(err)
		cc <- struct{}{}
	}
	<-cc
}
