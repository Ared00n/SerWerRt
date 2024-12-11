package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var dbUsers *sql.DB
var dbCandidates *sql.DB
var dbWorks *sql.DB

func InitDB() {
	var err error

	if _, err := os.Stat("db"); os.IsNotExist(err) {
		err = os.Mkdir("db", 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	dbUsers, err = sql.Open("sqlite3", "./db/users.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = dbUsers.Exec(`CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT UNIQUE,
        password TEXT
    )`)
	if err != nil {
		log.Fatal(err)
	}

	dbWorks, err = sql.Open("sqlite3", "./db/works.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = dbWorks.Exec(`CREATE TABLE IF NOT EXISTS works (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    informate TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    time_duration INTEGER NOT NULL,
    collaborators INTEGER NOT NULL,
    username TEXT NOT NULL,
    FOREIGN KEY (username) REFERENCES users (username)
)`)

	if err != nil {
		log.Fatal(err)
	}

	dbCandidates, err = sql.Open("sqlite3", "./db/candidates.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = dbCandidates.Exec(`CREATE TABLE IF NOT EXISTS candidates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    last_name TEXT,
    first_name TEXT,
    age INTEGER,
    profession TEXT,
    email TEXT,
    module INTEGER,
    username TEXT
)`)
	if err != nil {
		log.Fatal(err)
	}
}

func GetWorks() *sql.DB {
	return dbWorks
}

func GetDB() *sql.DB {
	return dbUsers
}

func GetCandidatesDB() *sql.DB {
	return dbCandidates
}

func CloseDB() {
	if dbUsers != nil {
		dbUsers.Close()
	}
	if dbCandidates != nil {
		dbCandidates.Close()
	}
	if dbWorks != nil {
		dbWorks.Close()
	}
}
