package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
)

type TokenRequest struct {
	Code string `json:"code"`
}

// handleToken exchanges a Discord OAuth2 authorization code for an access token.
// Served at /api/token; the frontend reaches it via the same /api proxy used
// for every other backend route.
func handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Exchange the code for an access_token
	resp, err := http.PostForm("https://discord.com/api/oauth2/token", url.Values{
		"client_id":     {os.Getenv("VITE_DISCORD_CLIENT_ID")},
		"client_secret": {os.Getenv("DISCORD_CLIENT_SECRET")},
		"grant_type":    {"authorization_code"},
		"code":          {req.Code},
	})
	if err != nil {
		log.Println("Discord token exchange error:", err)
		http.Error(w, "Failed to reach Discord", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Retrieve the access_token from the response
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		log.Println("Failed to parse Discord response:", err)
		http.Error(w, "Failed to parse Discord response", http.StatusBadGateway)
		return
	}

	// Return the access_token to our client as { access_token: "..." }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"access_token": body.AccessToken})
}
