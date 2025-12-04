//go:build devseed

package main

import "log"

func main() {
	ConnectSQL()
	SetupSQL()
	defer Database.Close()

	res, err := Database.Exec(`
		INSERT INTO servers (
			server_name,
			type,
			url,
			description,
			tags,
			logo_url,
			votes,
			added
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, "M1PPosu",
		0,
		"https://m1pposu.dev",
		"a osu! Server where we rank the unrankable! From HUGE Map Packs to Farm Maps, everything is rankable here, giving everyone and everything a chance to excel the rankings! With Vanilla, Relax and Autopilot leaderboards, you can never get bored!",
		"relax, autopilot, farm, diddy, 67",
		"/static/m1pplogo.png",
		0,
	)
	if err != nil {
		log.Fatal("insert server:", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Fatal("get server id:", err)
	}

	_, err = Database.Exec(`
		INSERT INTO users (username, discordid, server)
		VALUES (?, ?, ?)
	`, "M1PP Team", "123456789012345678", id)
	if err != nil {
		log.Fatal("insert user:", err)
	}

	log.Printf("Seeded server M1PPosu with id %d\n", id)
}
