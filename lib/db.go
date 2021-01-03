package lib

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type DBConfig struct {
	DSN string
}

func ConnectDB(dbConfig DBConfig) *sqlx.DB {
	db, err := sqlx.Connect("postgres", dbConfig.DSN)
	if err != nil {
		log.Fatalf("Error connecting to database: %s\n", err.Error())
	}
	return db
}

func ListenChannel(dbConfig DBConfig, channelName string, cb func(*pq.Notification)) {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	minReconn := 10 * time.Second
	maxReconn := time.Minute
	listener := pq.NewListener(dbConfig.DSN, minReconn, maxReconn, reportProblem)
	err := listener.Listen(channelName)
	if err != nil {
		panic(err)
	}
	for {
		cb(nil)

		for {
			select {
			case n := <-listener.Notify:
				cb(n)
			case <-time.After(90 * time.Second):
				// Received no events for a while, checking connection
				go listener.Ping()
			}
		}
	}
}
