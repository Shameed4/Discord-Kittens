package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type State struct {
	Counter int `json:"counter"`
	Players int `json:"players"`
}

type Lobby struct {
	Name    string
	Clients map[*websocket.Conn]bool
	Counter int
	Mutex   sync.Mutex
}

// Request payload for creating a lobby
type CreateLobbyRequest struct {
	Name string `json:"name"`
}

var (
	lobbies      = make(map[string]*Lobby)
	lobbiesMutex sync.Mutex

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// handleCreateLobby is our new HTTP REST endpoint
func handleCreateLobby(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the JSON body
	var req CreateLobbyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Lobby name cannot be empty", http.StatusBadRequest)
		return
	}

	lobbiesMutex.Lock()
	defer lobbiesMutex.Unlock()

	// Check if lobby already exists
	if _, exists := lobbies[req.Name]; exists {
		http.Error(w, "Lobby already exists", http.StatusConflict)
		return
	}

	// Create the new lobby
	lobbies[req.Name] = &Lobby{
		Name:    req.Name,
		Clients: make(map[*websocket.Conn]bool),
		Counter: 0,
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status": "created"}`))
	log.Printf("Lobby created: %s", req.Name)
}

func (l *Lobby) broadcast() {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	state := State{
		Counter: l.Counter,
		Players: len(l.Clients),
	}

	msg, err := json.Marshal(state)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	for client := range l.Clients {
		err := client.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			client.Close()
			delete(l.Clients, client)
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	lobbyName := r.URL.Query().Get("lobby")
	if lobbyName == "" {
		http.Error(w, "Missing lobby parameter", http.StatusBadRequest)
		return
	}

	// 1. Verify the lobby exists BEFORE upgrading to a WebSocket
	lobbiesMutex.Lock()
	lobby, exists := lobbies[lobbyName]
	lobbiesMutex.Unlock()

	if !exists {
		// Reject the connection with a standard HTTP 404
		http.Error(w, "Lobby not found. Create it first.", http.StatusNotFound)
		return
	}

	// 2. Upgrade the connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

	// 3. Join the lobby
	lobby.Mutex.Lock()
	lobby.Clients[ws] = true
	lobby.Mutex.Unlock()

	lobby.broadcast()

	// Listen for messages
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			lobby.Mutex.Lock()
			delete(lobby.Clients, ws)
			lobby.Mutex.Unlock()

			lobby.broadcast()

			lobbiesMutex.Lock()
			lobby.Mutex.Lock()
			if len(lobby.Clients) == 0 {
				delete(lobbies, lobbyName)
				log.Printf("Lobby deleted: %s", lobbyName)
			}
			lobby.Mutex.Unlock()
			lobbiesMutex.Unlock()

			break
		}

		lobby.Mutex.Lock()
		lobby.Counter++
		lobby.Mutex.Unlock()

		lobby.broadcast()
	}
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Register the new HTTP endpoint
	http.HandleFunc("/api/lobbies", handleCreateLobby)

	// Register the WebSocket endpoint
	http.HandleFunc("/ws", handleWebSocket)

	log.Println("Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
