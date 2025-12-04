//go:build !devseed
package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

type ServerResult struct {
	ID          int    `json:"id"`
	ServerName  string `json:"server_name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	LogoURL     string `json:"logo_url"`
	Status      string `json:"status"`
	Online      int    `json:"online"`
	Registered  int    `json:"registered"`
	Votes       int    `json:"votes"`
	Added       string `json:"added"`
	Owner       string `json:"owner"`
}

type ServerRequest struct {
	ID           int    `json:"id"`
	ServerName   string `json:"server_name"`
	URL          string `json:"url"`
	Description  string `json:"description"`
	Tags         string `json:"tags"`
	OwnerName    string `json:"owner_name"`
	OwnerDiscord string `json:"owner_discord"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

type turnstileResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

type SessionUser struct {
	DiscordID string `json:"discord_id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Expires   int64  `json:"exp"`
}

type DiscordConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scope        string
	APIBase      string
}

func verifyTurnstile(token, remoteIP string) bool {
	secret := strings.TrimSpace(os.Getenv("TURNSTILE_SECRET"))
	// dev / local: skip verification if no secret set :3
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

func loadDiscordConfig() (*DiscordConfig, error) {
	clientID := strings.TrimSpace(os.Getenv("DISCORD_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("DISCORD_CLIENT_SECRET"))
	redirectURI := strings.TrimSpace(os.Getenv("DISCORD_REDIRECT_URI"))

	if clientID == "" || clientSecret == "" || redirectURI == "" {
		return nil, fmt.Errorf("discord env not configured")
	}

	scope := strings.TrimSpace(os.Getenv("DISCORD_OAUTH_SCOPES"))
	if scope == "" {
		scope = "identify"
	}

	apiBase := strings.TrimSpace(os.Getenv("DISCORD_API_BASE"))
	if apiBase == "" {
		apiBase = "https://discord.com/api"
	}

	return &DiscordConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scope:        scope,
		APIBase:      apiBase,
	}, nil
}

func cookieSecure() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("SESSION_COOKIE_SECURE")))
	return v == "1" || v == "true" || v == "yes"
}

func generateStateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sessionSecret() []byte {
	s := strings.TrimSpace(os.Getenv("SESSION_SECRET"))
	if s == "" {
		// dev fallback – you'll wanna set a real value once ready for prod deployment lol.
		s = "dev-insecure-session-secret-change-me"
	}
	return []byte(s)
}

func encodeSessionToken(u SessionUser) (string, error) {
	if u.DiscordID == "" {
		return "", fmt.Errorf("empty discord id")
	}
	if u.Expires == 0 {
		u.Expires = time.Now().Add(30 * 24 * time.Hour).Unix()
	}

	payloadBytes, err := json.Marshal(u)
	if err != nil {
		return "", err
	}

	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, sessionSecret())
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	sigHex := hex.EncodeToString(sig)

	return payload + "." + sigHex, nil
}

func parseSessionToken(token string) (*SessionUser, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, false
	}
	payload, sigHex := parts[0], parts[1]

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return nil, false
	}

	mac := hmac.New(sha256.New, sessionSecret())
	mac.Write([]byte(payload))
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return nil, false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, false
	}

	var u SessionUser
	if err := json.Unmarshal(payloadBytes, &u); err != nil {
		return nil, false
	}
	if u.Expires != 0 && time.Now().Unix() > u.Expires {
		return nil, false
	}
	if u.DiscordID == "" {
		return nil, false
	}
	return &u, true
}

func main() {
	ConnectSQL()
	SetupSQL()
	defer Database.Close()

	app := fiber.New(fiber.Config{
		TrustProxy: true,
	})

	app.Use("/static", static.New("./public"))

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile("./public/index.html")
	})

	app.Get("/servers/:id", func(c fiber.Ctx) error {
		return c.SendFile("./public/server.html")
	})

	app.Get("/list", func(c fiber.Ctx) error {
		return c.SendFile("./public/list.html")
	})

	app.Get("/admin/requests", func(c fiber.Ctx) error {
		return c.SendFile("./public/admin_requests.html")
	})

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
	app.Post("/admin/requests/:id/approve", postAdminApproveHandler)
	app.Post("/admin/requests/:id/reject", postAdminRejectHandler)

	log.Fatal(app.Listen(":8080"))
}

/* Auth handlers */

func discordLoginHandler(c fiber.Ctx) error {
	cfg, err := loadDiscordConfig()
	if err != nil {
		log.Println("discordLoginHandler:", err)
		return c.Status(500).SendString("Discord login is not configured.")
	}

	state, err := generateStateToken()
	if err != nil {
		log.Println("generateStateToken:", err)
		return c.Status(500).SendString("Internal error.")
	}

	// stores state in http only cookie
	c.Cookie(&fiber.Cookie{
		Name:     "mossai_oauth_state",
		Value:    state,
		Path:     "/",
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		MaxAge:   600, // 10 minutes
	})

	authURL := fmt.Sprintf(
		"%s/oauth2/authorize?response_type=code&client_id=%s&scope=%s&redirect_uri=%s&state=%s",
		cfg.APIBase,
		url.QueryEscape(cfg.ClientID),
		url.QueryEscape(cfg.Scope),
		url.QueryEscape(cfg.RedirectURI),
		url.QueryEscape(state),
	)

	c.Set("Location", authURL)
    return c.SendStatus(fiber.StatusFound)
}

func discordCallbackHandler(c fiber.Ctx) error {
	if errStr := c.Query("error"); errStr != "" {
		return c.Status(400).SendString("Discord auth error: " + errStr)
	}

	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		return c.Status(400).SendString("Missing code or state.")
	}

	storedState := c.Cookies("mossai_oauth_state")
	if storedState == "" || storedState != state {
		return c.Status(400).SendString("Invalid state.")
	}

	// clear state cookie
	c.Cookie(&fiber.Cookie{
		Name:     "mossai_oauth_state",
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		MaxAge:   -1,
	})

	cfg, err := loadDiscordConfig()
	if err != nil {
		log.Println("discordCallbackHandler:", err)
		return c.Status(500).SendString("Discord login is not configured.")
	}

	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURI)

	req, err := http.NewRequest("POST", cfg.APIBase+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		log.Println("create token request:", err)
		return c.Status(500).SendString("Internal error.")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("token exchange:", err)
		return c.Status(500).SendString("Failed to contact Discord.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("token exchange status:", resp.StatusCode)
		return c.Status(500).SendString("Failed to exchange token.")
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Println("decode token response:", err)
		return c.Status(500).SendString("Failed to read Discord response.")
	}
	if tokenResp.AccessToken == "" {
		log.Println("empty access token")
		return c.Status(500).SendString("Failed to exchange token.")
	}

	// fetch user
	userReq, err := http.NewRequest("GET", cfg.APIBase+"/users/@me", nil)
	if err != nil {
		log.Println("create user request:", err)
		return c.Status(500).SendString("Internal error.")
	}
	userReq.Header.Set("Authorization", tokenResp.TokenType+" "+tokenResp.AccessToken)

	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		log.Println("user fetch:", err)
		return c.Status(500).SendString("Failed to contact Discord.")
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		log.Println("user fetch status:", userResp.StatusCode)
		return c.Status(500).SendString("Failed to fetch Discord user.")
	}

	var user struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		GlobalName string `json:"global_name"`
		Avatar     string `json:"avatar"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&user); err != nil {
		log.Println("decode user:", err)
		return c.Status(500).SendString("Failed to read Discord user.")
	}
	if user.ID == "" {
		log.Println("discord user without id")
		return c.Status(500).SendString("Invalid Discord user.")
	}

	displayName := user.GlobalName
	if displayName == "" {
		displayName = user.Username
	}

	var avatarURL string
	if user.Avatar != "" && user.ID != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", user.ID, user.Avatar)
	}

	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	session := SessionUser{
		DiscordID: user.ID,
		Username:  displayName,
		AvatarURL: avatarURL,
		Expires:   expiresAt.Unix(),
	}

	token, err := encodeSessionToken(session)
	if err != nil {
		log.Println("encodeSessionToken:", err)
		return c.Status(500).SendString("Internal error.")
	}

	    c.Cookie(&fiber.Cookie{
        Name:     "mossai_session",
        Value:    token,
        Path:     "/",
        HTTPOnly: true,
        Secure:   cookieSecure(),
        SameSite: "Lax",
        MaxAge:   int(time.Until(expiresAt).Seconds()),
    })

    // back to home ^ ^
    c.Set("Location", "/")
    return c.SendStatus(fiber.StatusFound)
}

func authMeHandler(c fiber.Ctx) error {
	token := c.Cookies("mossai_session")
	if token == "" {
		return c.JSON(fiber.Map{"authenticated": false})
	}

	user, ok := parseSessionToken(token)
	if !ok {
		return c.JSON(fiber.Map{"authenticated": false})
	}

	return c.JSON(fiber.Map{
		"authenticated": true,
		"discord_id":    user.DiscordID,
		"username":      user.Username,
		"avatar_url":    user.AvatarURL,
	})
}

func logoutHandler(c fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "mossai_session",
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		MaxAge:   -1,
	})
	return c.SendStatus(fiber.StatusNoContent)
}

// lb

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

// server detail

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

// voting

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

// list form - server_requests

func postServerRequestHandler(c fiber.Ctx) error {
	serverName := strings.TrimSpace(c.FormValue("server_name"))
	urlValue := strings.TrimSpace(c.FormValue("url"))
	description := strings.TrimSpace(c.FormValue("description"))
	tags := strings.TrimSpace(c.FormValue("tags"))
	ownerName := strings.TrimSpace(c.FormValue("owner_name"))
	ownerDiscord := strings.TrimSpace(c.FormValue("owner_discord"))
	tosAccepted := strings.TrimSpace(c.FormValue("tos_accept")) != ""
	captchaToken := c.FormValue("cf-turnstile-response")

	if serverName == "" || ownerName == "" || ownerDiscord == "" {
		return c.Status(400).SendString("server_name, owner_name and owner_discord are required")
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
			status,
			created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', datetime('now'))
	`, serverName, urlValue, description, tags, ownerName, ownerDiscord)
	if err != nil {
		log.Println("insert server_request:", err)
		return c.Status(500).SendString("internal error")
	}

	c.Set("Location", "/list?submitted=1")
	return c.SendStatus(fiber.StatusSeeOther)
}

// admin: list pending requests

func getAdminRequestsHandler(c fiber.Ctx) error {
	rows, err := Database.Query(`
		SELECT id,
		       server_name,
		       url,
		       description,
		       tags,
		       owner_name,
		       owner_discord,
		       status,
		       created_at
		FROM server_requests
		WHERE status = 'pending'
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Println("admin requests query error:", err)
		return c.Status(500).SendString("internal error")
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
		); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		requests = append(requests, r)
	}

	if err := rows.Err(); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.JSON(requests)
}

// admin: approve

func postAdminApproveHandler(c fiber.Ctx) error {
	id := c.Params("id", "0")

	var r ServerRequest
	err := Database.QueryRow(`
		SELECT server_name,
		       url,
		       description,
		       tags,
		       owner_name,
		       owner_discord
		FROM server_requests
		WHERE id = ? AND status = 'pending'
	`, id).Scan(
		&r.ServerName,
		&r.URL,
		&r.Description,
		&r.Tags,
		&r.OwnerName,
		&r.OwnerDiscord,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(404).SendString("request not found or already processed")
		}
		return c.Status(500).SendString(err.Error())
	}

	tx, err := Database.Begin()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO servers (
			server_name,
			type,
			url,
			description,
			tags,
			votes,
			added
		)
		VALUES (?, ?, ?, ?, ?, 0, datetime('now'))
	`, r.ServerName, 0, r.URL, r.Description, r.Tags)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	serverID, err := res.LastInsertId()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	if _, err := tx.Exec(`
		INSERT INTO users (username, discordid, server)
		VALUES (?, ?, ?)
	`, r.OwnerName, r.OwnerDiscord, serverID); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	if _, err := tx.Exec(`
		UPDATE server_requests
		SET status = 'approved'
		WHERE id = ?
	`, id); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	if err := tx.Commit(); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// admin: reject

func postAdminRejectHandler(c fiber.Ctx) error {
	id := c.Params("id", "0")

	res, err := Database.Exec(`
		UPDATE server_requests
		SET status = 'rejected'
		WHERE id = ? AND status = 'pending'
	`, id)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	if affected == 0 {
		return c.Status(404).SendString("request not found or already processed")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
