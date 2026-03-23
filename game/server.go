package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

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

func handleCreateLobby(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	lobby := new(Lobby)
	lobbies[req.Name] = lobby
	go lobby.run()

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status": "created"}`))
	log.Printf("Lobby created: %s", req.Name)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	lobbyName := r.URL.Query().Get("lobby")
	if lobbyName == "" {
		http.Error(w, "Missing lobby parameter", http.StatusBadRequest)
		return
	}

	// find if lobby exists
	lobbiesMutex.Lock()
	lobby, exists := lobbies[lobbyName]
	lobbiesMutex.Unlock()
	if !exists {
		http.Error(w, "Lobby not found. Create it first.", http.StatusNotFound)
		return
	}

	// request to join lobby
	gameStateChan := make(chan GameState)
	joinResultChan := make(chan JoinResponse)
	joinReq := JoinRequest{
		Send:   gameStateChan,
		Result: joinResultChan,
	}
	lobby.JoinQueue <- joinReq

	joinResponse := <-joinResultChan
	if !joinResponse.success {
		http.Error(w, "Lobby not found. Create it first.", http.StatusNotFound)
		return
	}
	playerId := joinResponse.playerId

	// upgrade the connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

	// send messages to client
	go func() {
		for state := range gameStateChan {
			ws.WriteJSON(state)
		}
	}()

	// send client updates to lobby
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			log.Println("Player disconnected or read error")
			break
		}
		var action PlayerAction
		json.Unmarshal(data, &action)
		action.playerId = playerId
		lobby.ActionQueue <- action
	}

	quitAction := PlayerAction{
		playerId:   playerId,
		actionType: Quit,
	}
	lobby.ActionQueue <- quitAction
}

func main() {
	http.HandleFunc("/lobby", handleCreateLobby)
	http.HandleFunc("/ws", handleWebSocket)
}
