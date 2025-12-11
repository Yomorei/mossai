package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type turnstileResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

func verifyTurnstile(token, remoteIP string) bool {
	secret := strings.TrimSpace(os.Getenv("TURNSTILE_SECRET"))
	if secret == "" {
		return true
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}

	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", form)
	if err != nil {
		log.Println("turnstile verify error:", err)
		return false
	}
	defer resp.Body.Close()

	var parsed turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		log.Println("turnstile decode error:", err)
		return false
	}
	if !parsed.Success {
		log.Println("turnstile failed:", parsed.ErrorCodes)
	}
	return parsed.Success
}

func getLeaderboardHandler(c fiber.Ctx) error {
	rows, err := Database.Query(`
		SELECT s.id,
		       s.server_name,
		       COALESCE(s.url, ''),
		       COALESCE(s.description, ''),
		       COALESCE(s.tags, ''),
		       COALESCE(s.logo_url, ''),
		       COALESCE(s.status, 'unknown'),
		       COALESCE(s.online, 0),
		       COALESCE(s.registered, 0),
		       s.votes,
		       s.added,
		       u.username
		FROM servers s
		JOIN users u
		  ON u.server = s.id
		ORDER BY s.votes DESC, s.added DESC
	`)
	if err != nil {
		log.Println("leaderboard query error:", err)
		return c.Status(500).SendString("internal error")
	}
	defer rows.Close()

	servers := make([]ServerResult, 0, 16)

	for rows.Next() {
		var s ServerResult
		if err := rows.Scan(
			&s.ID,
			&s.ServerName,
			&s.URL,
			&s.Description,
			&s.Tags,
			&s.LogoURL,
			&s.Status,
			&s.Online,
			&s.Registered,
			&s.Votes,
			&s.Added,
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
}

func getServerHandler(c fiber.Ctx) error {
	id := c.Params("id", "0")

	row := Database.QueryRow(`
		SELECT s.id,
		       s.server_name,
		       COALESCE(s.url, ''),
		       COALESCE(s.description, ''),
		       COALESCE(s.tags, ''),
		       COALESCE(s.logo_url, ''),
		       COALESCE(s.status, 'unknown'),
		       COALESCE(s.online, 0),
		       COALESCE(s.registered, 0),
		       s.votes,
		       s.added,
		       u.username
		FROM servers s
		JOIN users u
		  ON u.server = s.id
		WHERE s.id = ?
	`, id)

	var s ServerResult

	if err := row.Scan(
		&s.ID,
		&s.ServerName,
		&s.URL,
		&s.Description,
		&s.Tags,
		&s.LogoURL,
		&s.Status,
		&s.Online,
		&s.Registered,
		&s.Votes,
		&s.Added,
		&s.Owner,
	); err != nil {
		if err == sql.ErrNoRows {
			return c.Status(404).SendString("server not found")
		}
		return c.Status(500).SendString(err.Error())
	}

	return c.JSON(s)
}

func postVoteHandler(c fiber.Ctx) error {
	id := c.Params("id", "0")
	ip := c.IP()
	name := c.FormValue("name")

	if strings.TrimSpace(name) == "" {
		return c.Status(400).SendString("name is required")
	}

	var lastVote string
	err := Database.QueryRow(`
		SELECT last_vote
		FROM votes
		WHERE server = ? AND ip = ?
		  AND last_vote > datetime('now', '-12 hours')
		ORDER BY last_vote DESC
		LIMIT 1
	`, id, ip).Scan(&lastVote)

	if err != nil && err != sql.ErrNoRows {
		return c.Status(500).SendString(err.Error())
	}

	if err == nil {
		return c.Status(429).SendString("You can vote for this server again in 12 hours.")
	}

	if _, err := Database.Exec(
		"UPDATE servers SET votes = votes + 1 WHERE id = ?",
		id,
	); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	if _, err := Database.Exec(
		`INSERT INTO votes (ip, server, user_name, last_vote)
		 VALUES (?, ?, ?, datetime('now'))`,
		ip,
		id,
		name,
	); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.SendString("ok")
}

func postServerRequestHandler(c fiber.Ctx) error {
	serverName := strings.TrimSpace(c.FormValue("server_name"))
	urlValue := strings.TrimSpace(c.FormValue("url"))
	description := strings.TrimSpace(c.FormValue("description"))
	tags := strings.TrimSpace(c.FormValue("tags"))
	ownerName := strings.TrimSpace(c.FormValue("owner_name"))
	ownerDiscord := strings.TrimSpace(c.FormValue("owner_discord"))
	logoURL := strings.TrimSpace(c.FormValue("logo_url"))
	tosAccepted := strings.TrimSpace(c.FormValue("tos_accept")) != ""
	captchaToken := c.FormValue("cf-turnstile-response")

	if serverName == "" || ownerName == "" || ownerDiscord == "" {
		return c.Status(400).SendString("server_name, owner_name and owner_discord are required")
	}

	if len(description) > MaxDescriptionLength {
		return c.Status(400).SendString(
			fmt.Sprintf("Description must be at most %d characters.", MaxDescriptionLength),
		)
	}

	if !tosAccepted {
		return c.Status(400).SendString("You must accept the Terms of Service to submit.")
	}

	if !verifyTurnstile(captchaToken, c.IP()) {
		return c.Status(400).SendString("Captcha verification failed.")
	}

	_, err := Database.Exec(`
		INSERT INTO server_requests (
			server_name,
			url,
			description,
			tags,
			owner_name,
			owner_discord,
			logo_url,
			status,
			created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', datetime('now'))
	`, serverName, urlValue, description, tags, ownerName, ownerDiscord, logoURL)
	if err != nil {
		log.Println("insert server_request:", err)
		return c.Status(500).SendString("internal error")
	}

	req := ServerRequest{
		ServerName:   serverName,
		URL:          urlValue,
		Description:  description,
		Tags:         tags,
		OwnerName:    ownerName,
		OwnerDiscord: ownerDiscord,
		LogoURL:      logoURL,
		Status:       "pending",
	}
	notifyNewServerRequest(&req)

	c.Set("Location", "/list?submitted=1")
	return c.SendStatus(fiber.StatusSeeOther)
}
