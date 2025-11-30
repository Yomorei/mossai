package main

import (
	"database/sql"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var Database *sql.DB

func ConnectSQL() {
	database, err := sql.Open("sqlite3", "file:mossai.db?_journal=WAL&_synchronous=NORMAL&_foreign_keys=ON&_temp_store=MEMORY&_mmap_size=268435456")

	if err != nil {
		panic(err)
	}

	Database = database
}

func SetupSQL() {
	if _, err := Database.Exec(`CREATE TABLE IF NOT EXISTS "servers" (
		"id"			INTEGER NOT NULL,
		"server_name"	TEXT NOT NULL,
		"type"			INTEGER NOT NULL,
		"votes"			INTEGER NOT NULL DEFAULT 0,
		"added"			DATETIME NOT NULL,
		PRIMARY KEY("id" AUTOINCREMENT)
	);
`); err != nil {
		panic(err)
	}

	if _, err := Database.Exec(`
	CREATE TABLE IF NOT EXISTS "votes" (
		"server"	INTEGER NOT NULL,
		"ip"		TEXT NOT NULL,
		"last_vote"	DATETIME NOT NULL,
		FOREIGN KEY("server") REFERENCES "servers"("id")
	);
	`); err != nil {
		panic(err)
	}

	if _, err := Database.Exec(`
	CREATE TABLE IF NOT EXISTS "users" (
		"id"        INTEGER NOT NULL,
		"username"  TEXT NOT NULL,
		"discordid" TEXT NOT NULL,
		"server"    INTEGER,
		PRIMARY KEY("id" AUTOINCREMENT),
		FOREIGN KEY("server") REFERENCES "servers"("id")
	);
	`); err != nil {
		panic(err)
	}
}
