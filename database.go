package main

import (
	"database/sql"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var Database *sql.DB

func ConnectSQL() {
	db, err := sql.Open(
		"sqlite3",
		"file:mossai.db?_journal=WAL&_synchronous=NORMAL&_foreign_keys=ON&_temp_store=MEMORY&_mmap_size=268435456",
	)
	if err != nil {
		panic(err)
	}

	Database = db
}

func SetupSQL() {
	if _, err := Database.Exec(`
		CREATE TABLE IF NOT EXISTS servers (
			id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			server_name TEXT    NOT NULL,
			type        INTEGER NOT NULL,
			url         TEXT,
			description TEXT,
			tags        TEXT,
			logo_url    TEXT,
			status      TEXT    NOT NULL DEFAULT 'unknown',
			online      INTEGER,
			registered  INTEGER,
			votes       INTEGER NOT NULL DEFAULT 0,
			added       DATETIME NOT NULL
		);
	`); err != nil {
		panic(err)
	}

	if _, err := Database.Exec(`
		CREATE TABLE IF NOT EXISTS votes (
			server    INTEGER  NOT NULL,
			ip        TEXT     NOT NULL,
			user_name TEXT     NOT NULL,
			last_vote DATETIME NOT NULL,
			FOREIGN KEY(server) REFERENCES servers(id)
		);
	`); err != nil {
		panic(err)
	}

	if _, err := Database.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id        INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			username  TEXT    NOT NULL,
			discordid TEXT    NOT NULL,
			server    INTEGER,
			FOREIGN KEY(server) REFERENCES servers(id)
		);
	`); err != nil {
		panic(err)
	}

	if _, err := Database.Exec(`
		CREATE TABLE IF NOT EXISTS server_requests (
			id            INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			server_name   TEXT    NOT NULL,
			url           TEXT,
			description   TEXT,
			tags          TEXT,
			owner_name    TEXT    NOT NULL,
			owner_discord TEXT    NOT NULL,
			status        TEXT    NOT NULL DEFAULT 'pending',
			created_at    DATETIME NOT NULL
		);
	`); err != nil {
		panic(err)
	}
}
