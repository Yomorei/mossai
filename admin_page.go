package main

import "github.com/gofiber/fiber/v3"

func adminPageHandler(c fiber.Ctx) error {
	u, ok := getSessionUser(c)
	if !ok || !isAdminDiscordID(u.DiscordID) {
		c.Set("Location", "/")
		return c.SendStatus(fiber.StatusFound)
	}
	return c.SendFile("./public/admin_requests.html")
}
