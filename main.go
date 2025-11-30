package main

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

type ServerResult struct {
	ID         int    `json:"id"`
	ServerName string `json:"server_name"`
	URL        string `json:"url"` 
	Votes      int    `json:"votes"`
	Added      string `json:"added"`
	Owner      string `json:"owner"`
}

func main() {
	ConnectSQL()
	SetupSQL()

	app := fiber.New(fiber.Config{
		TrustProxy: true,
	})

	// Y
	app.Use("/static/*", static.New("./public"))

	// Y
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile("./public/index.html")
	})

	// LB api type shi :3
	app.Get("/leaderboard", func(c fiber.Ctx) error {
		rows, err := Database.Query(`
			SELECT s.id,
			       s.server_name,
			       s.votes,
			       s.added,
			       '' as url,
			       u.username
			FROM servers s
			JOIN users u
			  ON u.server = s.id
			ORDER BY s.votes DESC
		`)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		defer rows.Close()

		var servers []ServerResult

		for rows.Next() {
			var s ServerResult
			if err := rows.Scan(
				&s.ID,
				&s.ServerName,
				&s.Votes,
				&s.Added,
				&s.URL,
				&s.Owner,
			); err != nil {
				return c.Status(500).SendString(err.Error())
			}
			servers = append(servers, s)
		}

		if err := rows.Err(); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(servers)
	})

	app.Get("/server/:id", func(c fiber.Ctx) error {
		row := Database.QueryRow(`
			SELECT s.id,
			       s.server_name,
			       s.votes,
			       s.added,
			       '' as url,
			       u.username
			FROM servers s
			JOIN users u
			  ON u.server = s.id
			WHERE s.id = ?
		`, c.Params("id", "0"))

		var s ServerResult

		if err := row.Scan(
			&s.ID,
			&s.ServerName,
			&s.Votes,
			&s.Added,
			&s.URL,
			&s.Owner,
		); err != nil {
			return c.Status(404).SendString("server not found")
		}

		return c.JSON(s)
	})

	app.Post("/server/:id/vote", func(c fiber.Ctx) error {
		id := c.Params("id", "0")

		if _, err := Database.Exec(
			"UPDATE servers SET votes = votes + 1 WHERE id = ?",
			id,
		); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		if _, err := Database.Exec(
			`INSERT INTO votes (ip, server, last_vote)
			 VALUES (?, ?, datetime('now'))`,
			c.IP(),
			id,
		); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.SendString("ok")
	})

	app.Listen(":8080")
}
