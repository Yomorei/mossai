package main

import (
	"database/sql"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func getAdminRequestsHandler(c fiber.Ctx) error {
	if _, err := requireAdmin(c); err != nil {
		return err
	}

	rows, err := Database.Query(`
		SELECT
			id,
			server_name,
			COALESCE(url, ''),
			COALESCE(description, ''),
			COALESCE(tags, ''),
			owner_name,
			owner_discord,
			status,
			created_at,
			COALESCE(logo_url, '')
		FROM server_requests
		WHERE status = 'pending'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to load requests")
	}
	defer rows.Close()

	requests := make([]ServerRequest, 0, 16)

	for rows.Next() {
		var r ServerRequest
		if err := rows.Scan(
			&r.ID,
			&r.ServerName,
			&r.URL,
			&r.Description,
			&r.Tags,
			&r.OwnerName,
			&r.OwnerDiscord,
			&r.Status,
			&r.CreatedAt,
			&r.LogoURL,
		); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("failed to scan request")
		}
		requests = append(requests, r)
	}

	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to read requests")
	}

	return c.JSON(requests)
}

func postAdminUpdateHandler(c fiber.Ctx) error {
	if _, err := requireAdmin(c); err != nil {
		return err
	}

	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).SendString("missing id")
	}

	type updatePayload struct {
		ServerName   string   `json:"server_name"`
		URL          string   `json:"url"`
		LogoURL      string   `json:"logo_url"`
		Description  string   `json:"description"`
		Tags         []string `json:"tags"`
		OwnerName    string   `json:"owner_name"`
		OwnerDiscord string   `json:"owner_discord"`
	}

	var payload updatePayload
	if err := c.Bind().Body(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid json body")
	}

	payload.ServerName = strings.TrimSpace(payload.ServerName)
	payload.URL = strings.TrimSpace(payload.URL)
	payload.LogoURL = strings.TrimSpace(payload.LogoURL)
	payload.Description = strings.TrimSpace(payload.Description)
	payload.OwnerName = strings.TrimSpace(payload.OwnerName)
	payload.OwnerDiscord = strings.TrimSpace(payload.OwnerDiscord)

	if payload.ServerName == "" || payload.OwnerName == "" || payload.OwnerDiscord == "" {
		return c.Status(fiber.StatusBadRequest).SendString("server_name, owner_name and owner_discord are required")
	}

	var tagsJoined string
	if len(payload.Tags) > 0 {
		clean := make([]string, 0, len(payload.Tags))
		for _, t := range payload.Tags {
			t = strings.TrimSpace(t)
			if t != "" {
				clean = append(clean, t)
			}
		}
		tagsJoined = strings.Join(clean, ",")
	}

	res, err := Database.Exec(`
		UPDATE server_requests
		SET
			server_name   = ?,
			url           = ?,
			logo_url      = ?,
			description   = ?,
			tags          = ?,
			owner_name    = ?,
			owner_discord = ?
		WHERE id = ? AND status = 'pending'
	`,
		payload.ServerName,
		nullEmpty(payload.URL),
		nullEmpty(payload.LogoURL),
		nullEmpty(payload.Description),
		nullEmpty(tagsJoined),
		payload.OwnerName,
		payload.OwnerDiscord,
		id,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to update request")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to read update result")
	}
	if affected == 0 {
		return c.Status(fiber.StatusNotFound).SendString("request not found or not pending")
	}

	return c.JSON(fiber.Map{"ok": true})
}

func postAdminApproveHandler(c fiber.Ctx) error {
	if _, err := requireAdmin(c); err != nil {
		return err
	}

	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).SendString("missing id")
	}

	var r ServerRequest
	err := Database.QueryRow(`
		SELECT
			id,
			server_name,
			COALESCE(url, ''),
			COALESCE(description, ''),
			COALESCE(tags, ''),
			owner_name,
			owner_discord,
			status,
			created_at,
			COALESCE(logo_url, '')
		FROM server_requests
		WHERE id = ? AND status = 'pending'
	`, id).Scan(
		&r.ID,
		&r.ServerName,
		&r.URL,
		&r.Description,
		&r.Tags,
		&r.OwnerName,
		&r.OwnerDiscord,
		&r.Status,
		&r.CreatedAt,
		&r.LogoURL,
	)
	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusNotFound).SendString("request not found or already processed")
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to load request")
	}

	tx, err := Database.Begin()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to begin transaction")
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO servers (
			server_name,
			type,
			url,
			description,
			tags,
			logo_url,
			status,
			votes,
			added
		)
		VALUES (?, ?, ?, ?, ?, ?, 'unknown', 0, datetime('now'))
	`,
		r.ServerName,
		0,
		nullEmpty(r.URL),
		nullEmpty(r.Description),
		nullEmpty(r.Tags),
		nullEmpty(r.LogoURL),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to create server")
	}

	serverID, err := res.LastInsertId()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get server id")
	}

	_, err = tx.Exec(`
		INSERT INTO users (username, discordid, server)
		VALUES (?, ?, ?)
	`,
		r.OwnerName,
		r.OwnerDiscord,
		serverID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to create owner user")
	}

	_, err = tx.Exec(`
		UPDATE server_requests
		SET status = 'approved'
		WHERE id = ?
	`, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to update request status")
	}

	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to commit transaction")
	}

	notifyServerRequestApproved(&r, serverID)

	return c.JSON(fiber.Map{
		"ok":        true,
		"server_id": serverID,
	})
}

func postAdminRejectHandler(c fiber.Ctx) error {
	if _, err := requireAdmin(c); err != nil {
		return err
	}

	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).SendString("missing id")
	}

	var r ServerRequest
	err := Database.QueryRow(`
		SELECT
			id,
			server_name,
			COALESCE(url, ''),
			COALESCE(description, ''),
			COALESCE(tags, ''),
			owner_name,
			owner_discord,
			status,
			created_at,
			COALESCE(logo_url, '')
		FROM server_requests
		WHERE id = ? AND status = 'pending'
	`, id).Scan(
		&r.ID,
		&r.ServerName,
		&r.URL,
		&r.Description,
		&r.Tags,
		&r.OwnerName,
		&r.OwnerDiscord,
		&r.Status,
		&r.CreatedAt,
		&r.LogoURL,
	)
	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusNotFound).SendString("request not found or already processed")
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to load request")
	}

	res, err := Database.Exec(`
		UPDATE server_requests
		SET status = 'rejected'
		WHERE id = ? AND status = 'pending'
	`, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to update request status")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to read update result")
	}
	if affected == 0 {
		return c.Status(fiber.StatusNotFound).SendString("request not found or already processed")
	}

	notifyServerRequestRejected(&r)

	return c.JSON(fiber.Map{"ok": true})
}

func postAdminRemoveServerHandler(c fiber.Ctx) error {
	if _, err := requireAdmin(c); err != nil {
		return err
	}

	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).SendString("missing id")
	}

	res, err := Database.Exec(`
		DELETE FROM servers
		WHERE id = ?
	`, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to remove server")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to read delete result")
	}
	if affected == 0 {
		return c.Status(fiber.StatusNotFound).SendString("server not found")
	}

	return c.JSON(fiber.Map{"ok": true})
}

func nullEmpty(s string) string {
	return strings.TrimSpace(s)
}
