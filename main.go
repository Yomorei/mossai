package main

import "github.com/gofiber/fiber/v3"

type ServerResult struct {
	ServerName string  `json:"server_name"`
	URL        *string `json:"url"`
	Votes      int     `json:"votes"`
	Added      string  `json:"added"`
	Owner      string  `json:"owner"`
}

func main() {
	ConnectSQL()
	SetupSQL()
	app := fiber.New(fiber.Config{TrustProxy: true})

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hi :3")
	})

	app.Get("/leaderboard", func(c fiber.Ctx) error {
		rows, err := Database.Query("SELECT s.server_name, s.votes, s.added, s.url, u.username FROM servers s JOIN users u WHERE u.server = s.id ORDER BY votes DESC")

		if err != nil {
			return nil
		}

		defer rows.Close()

		var servers []ServerResult

		for rows.Next() {
			var s ServerResult
			if err := rows.Scan(
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
		row := Database.QueryRow("SELECT s.server_name, s.votes, s.added, s.url, u.username FROM servers s JOIN users u WHERE u.server = s.id AND s.id = ?", c.Params("id", "0"))

		var s ServerResult

		if err := row.Scan(
			&s.ServerName,
			&s.Votes,
			&s.Added,
			&s.URL,
			&s.Owner,
		); err != nil {
			return nil
		}
		return c.JSON(s)
	})

	app.Post("/server/:id/vote", func(c fiber.Ctx) error {
		if _, err := Database.Exec("UPDATE servers SET votes = votes + 1 WHERE id = ?", c.Params("id", "0")); err != nil {
			return err
		}

		if _, err := Database.Exec(`INSERT INTO votes (ip, server, last_vote) VALUES (?, ?, datetime('now'))`, c.IP(), c.Params("id")); err != nil {
			return err
		}

		return c.SendString("ok")
	})

	app.Listen(":8080")
}
