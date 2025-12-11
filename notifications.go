package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordFooter struct {
	Text string `json:"text,omitempty"`
}

type discordThumbnail struct {
	URL string `json:"url,omitempty"`
}

type discordAuthor struct {
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type discordEmbed struct {
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	URL         string            `json:"url,omitempty"`
	Color       int               `json:"color,omitempty"`
	Fields      []discordField    `json:"fields,omitempty"`
	Timestamp   string            `json:"timestamp,omitempty"`
	Footer      *discordFooter    `json:"footer,omitempty"`
	Thumbnail   *discordThumbnail `json:"thumbnail,omitempty"`
	Author      *discordAuthor    `json:"author,omitempty"`
}

type discordWebhookPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []discordEmbed `json:"embeds,omitempty"`
}

func getAdminWebhookURL() string {
	return strings.TrimSpace(os.Getenv("DISCORD_ADMIN_WEBHOOK_URL"))
}

func adminMention() string {
	if v := strings.TrimSpace(os.Getenv("DISCORD_ADMIN_PING")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("DISCORD_ADMIN_MENTION")); v != "" {
		return v
	}
	return ""
}

func sendAdminWebhook(embed discordEmbed) {
	webhookURL := getAdminWebhookURL()
	if webhookURL == "" {
		return
	}

	payload := discordWebhookPayload{
		Embeds: []discordEmbed{embed},
	}
	if mention := adminMention(); mention != "" {
		payload.Content = mention
	}

	buf, err := json.Marshal(payload)
	if err != nil {
		log.Println("sendAdminWebhook marshal:", err)
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(buf))
	if err != nil {
		log.Println("sendAdminWebhook post:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Println("sendAdminWebhook status:", resp.StatusCode)
	}
}

func notifyNewServerRequest(r *ServerRequest) {
	descRaw := coalesce(r.Description, "No description was provided.")
	descSnippet := truncate(descRaw, 200)

	desc := "A new osu! private server has been submitted and is waiting in the review queue.\n\n"
	if descSnippet != "" {
		desc += fmt.Sprintf("> %s", descSnippet)
	}

	var thumb *discordThumbnail
	if url := strings.TrimSpace(r.LogoURL); url != "" {
		thumb = &discordThumbnail{URL: url}
	}

	embed := discordEmbed{
		Title:       fmt.Sprintf("🆕 New server request · %s", coalesce(r.ServerName, "unnamed server")),
		Description: desc,
		Color:       0xFF66AA,
		Author: &discordAuthor{
			Name: coalesce(r.ServerName, "unnamed server"),
		},
		Fields: []discordField{
			{
				Name:   "Owner",
				Value:  formatDiscordOwner(r.OwnerName, r.OwnerDiscord),
				Inline: true,
			},
			{
				Name:   "Request ID",
				Value:  fmt.Sprintf("`%d`", r.ID),
				Inline: true,
			},
			{
				Name:   "Submitted at",
				Value:  coalesce(r.CreatedAt, "N/A"),
				Inline: true,
			},
			{
				Name:   "Server URL",
				Value:  coalesce(r.URL, "N/A"),
				Inline: false,
			},
			{
				Name:   "Tags",
				Value:  formatTagsCSV(r.Tags),
				Inline: false,
			},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Footer: &discordFooter{
			Text: footerText("new server request"),
		},
		Thumbnail: thumb,
	}

	sendAdminWebhook(embed)
}

func notifyServerRequestApproved(r *ServerRequest, serverID int64) {
	desc := "The request has been approved and the server is now live on mossai."

	var thumb *discordThumbnail
	if url := strings.TrimSpace(r.LogoURL); url != "" {
		thumb = &discordThumbnail{URL: url}
	}

	serverURL := buildServerURL(serverID)

	embed := discordEmbed{
		Title:       "🟢 Server approved",
		Description: desc,
		URL:         serverURL,
		Color:       0x57F287,
		Author: &discordAuthor{
			Name: coalesce(r.ServerName, "unnamed server"),
			URL:  serverURL,
		},
		Fields: []discordField{
			{
				Name:   "Server ID",
				Value:  fmt.Sprintf("`%d`", serverID),
				Inline: true,
			},
			{
				Name:   "Request ID",
				Value:  fmt.Sprintf("`%d`", r.ID),
				Inline: true,
			},
			{
				Name:   "Owner",
				Value:  formatDiscordOwner(r.OwnerName, r.OwnerDiscord),
				Inline: false,
			},
			{
				Name:   "Server URL",
				Value:  coalesce(r.URL, "N/A"),
				Inline: false,
			},
			{
				Name:   "Tags",
				Value:  formatTagsCSV(r.Tags),
				Inline: false,
			},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Footer: &discordFooter{
			Text: footerText("server approved"),
		},
		Thumbnail: thumb,
	}

	sendAdminWebhook(embed)
}

func notifyServerRequestRejected(r *ServerRequest) {
	descRaw := coalesce(r.Description, "No description was provided.")
	descSnippet := truncate(descRaw, 200)

	desc := "The server request was rejected.\n\n"
	if descSnippet != "" {
		desc += fmt.Sprintf("> %s", descSnippet)
	}

	embed := discordEmbed{
		Title:       "⛔ Server request rejected",
		Description: desc,
		Color:       0xED4245, // red
		Author: &discordAuthor{
			Name: coalesce(r.ServerName, "unnamed server"),
		},
		Fields: []discordField{
			{
				Name:   "Owner",
				Value:  formatDiscordOwner(r.OwnerName, r.OwnerDiscord),
				Inline: true,
			},
			{
				Name:   "Request ID",
				Value:  fmt.Sprintf("`%d`", r.ID),
				Inline: true,
			},
			{
				Name:   "Submitted at",
				Value:  coalesce(r.CreatedAt, "N/A"),
				Inline: false,
			},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Footer: &discordFooter{
			Text: footerText("server rejected"),
		},
	}

	sendAdminWebhook(embed)
}

func coalesce(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func truncate(s string, limit int) string {
	s = strings.TrimSpace(s)
	if s == "" || limit <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	if limit <= 1 {
		return string(runes[:1])
	}
	return string(runes[:limit-1]) + "…"
}

func formatTagsCSV(tags string) string {
	tags = strings.TrimSpace(tags)
	if tags == "" {
		return "none"
	}
	parts := strings.Split(tags, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, fmt.Sprintf("`%s`", p))
		}
	}
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, " · ")
}

func formatDiscordOwner(name, discordID string) string {
	name = coalesce(name, "unknown")
	discordID = strings.TrimSpace(discordID)
	if discordID == "" {
		return name
	}
	if _, err := strconv.ParseInt(discordID, 10, 64); err == nil {
		return fmt.Sprintf("%s (<@%s>)", name, discordID)
	}
	return fmt.Sprintf("%s (%s)", name, discordID)
}

func mossaiEnvLabel() string {
	if v := strings.TrimSpace(os.Getenv("MOSSAI_ENV")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("APP_ENV")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("ENV")); v != "" {
		return v
	}
	return "local"
}

func footerText(context string) string {
	return fmt.Sprintf("mossai · %s · %s", context, mossaiEnvLabel())
}

func getBaseURL() string {
	base := strings.TrimSpace(os.Getenv("MOSSAI_BASE_URL"))
	if base == "" {
		base = strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL"))
	}
	if base == "" {
		base = "http://localhost:8080"
	}
	return strings.TrimRight(base, "/")
}

func buildServerURL(id int64) string {
	return fmt.Sprintf("%s/servers/%d", getBaseURL(), id)
}
