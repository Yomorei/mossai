package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
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
)

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

func sessionSecret() []byte {
	s := strings.TrimSpace(os.Getenv("SESSION_SECRET"))
	if s == "" {
		s = "dev-insecure-session-secret-change-me"
	}
	return []byte(s)
}

func generateStateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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

func getSessionUser(c fiber.Ctx) (*SessionUser, bool) {
	token := c.Cookies("mossai_session")
	if token == "" {
		return nil, false
	}
	u, ok := parseSessionToken(token)
	return u, ok
}

func parseAdminEnvIDs() []string {
	raw := strings.TrimSpace(os.Getenv("MOSS_ADMIN_IDS"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		id := strings.TrimSpace(p)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func isAdminDiscordID(id string) bool {
	if id == "" {
		return false
	}
	for _, adminID := range parseAdminEnvIDs() {
		if adminID == id {
			return true
		}
	}
	return false
}

func requireAdmin(c fiber.Ctx) (*SessionUser, error) {
	u, ok := getSessionUser(c)
	if !ok {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if !isAdminDiscordID(u.DiscordID) {
		return nil, fiber.NewError(fiber.StatusForbidden, "forbidden")
	}
	return u, nil
}

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

	c.Cookie(&fiber.Cookie{
		Name:     "mossai_oauth_state",
		Value:    state,
		Path:     "/",
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		MaxAge:   600,
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

	c.Set("Location", "/")
	return c.SendStatus(fiber.StatusFound)
}

func authMeHandler(c fiber.Ctx) error {
	u, ok := getSessionUser(c)
	if !ok {
		return c.JSON(fiber.Map{"authenticated": false})
	}

	return c.JSON(fiber.Map{
		"authenticated": true,
		"discord_id":    u.DiscordID,
		"username":      u.Username,
		"avatar_url":    u.AvatarURL,
		"is_admin":      isAdminDiscordID(u.DiscordID),
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
