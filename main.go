//go:build !devseed

package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func main() {
	ConnectSQL()
	SetupSQL()
	defer Database.Close()

	app := fiber.New(fiber.Config{
		TrustProxy: true,
	})

	// static
	app.Use("/static", static.New("./public"))

	// pages
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile("./public/index.html")
	})
	app.Get("/servers/:id", func(c fiber.Ctx) error {
		return c.SendFile("./public/server.html")
	})
	app.Get("/list", func(c fiber.Ctx) error {
		return c.SendFile("./public/list.html")
	})
	app.Get("/admin/requests", adminPageHandler)

	// public JSON APIs
	app.Get("/leaderboard", getLeaderboardHandler)
	app.Get("/server/:id", getServerHandler)
	app.Post("/server/:id/vote", postVoteHandler)
	app.Post("/list", postServerRequestHandler)

	// auth APIs
	app.Get("/auth/discord/login", discordLoginHandler)
	app.Get("/auth/discord/callback", discordCallbackHandler)
	app.Get("/auth/me", authMeHandler)
	app.Post("/auth/logout", logoutHandler)

	// admin JSON APIs
	app.Get("/admin/requests/data", getAdminRequestsHandler)
	app.Post("/admin/requests/:id/update", postAdminUpdateHandler)
	app.Post("/admin/requests/:id/approve", postAdminApproveHandler)
	app.Post("/admin/requests/:id/reject", postAdminRejectHandler)
	app.Post("/api/admin/servers/:id/remove", postAdminRemoveServerHandler)
	log.Fatal(app.Listen(":8080"))
}
