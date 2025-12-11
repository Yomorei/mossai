package main

const MaxDescriptionLength = 250

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
	LogoURL      string `json:"logo_url"`
}
