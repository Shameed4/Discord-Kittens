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

type ActionRequest struct {
	ActionStr string `json:"action"`
	Index     int    `json:"index"`
}

var (
	lobbies      = make(map[string]*Lobby)
	lobbiesMutex sync.Mutex

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func handleCreateLobby(w http.ResponseWriter, r *http.Request) {
	log.Println("Requested to create lobby")
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

	lobby := NewLobby()
	lobbies[req.Name] = lobby
	go lobby.run()

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status": "created"}`))
	log.Printf("Lobby created: %s", req.Name)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Println("New request to join lobby")
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

	// upgrade the connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

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
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4000, joinResponse.error))
		return
	}
	playerId := joinResponse.playerId
	log.Printf("Player %d successfully joined lobby %s", playerId, lobbyName)

	// send messages to client
	go func() {
		for state := range gameStateChan {
			if err := ws.WriteJSON(state); err != nil {
				log.Println("Write error:", err)
				break
			}
		}
	}()

	// send client updates to lobby
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Player %d disconnected lobby %s or read error ", playerId, lobbyName)
			break
		}
		var actionRequest ActionRequest
		if err := json.Unmarshal(data, &actionRequest); err != nil {
			lobby.sendError(playerId, "Failed to parse request")
			continue
		}

		log.Printf("Action request %+v", actionRequest)
		actionType, ok := actionTypeNames[actionRequest.ActionStr]
		if !ok {
			lobby.sendError(playerId, "Invalid action string")
			continue
		}

		if actionType == Disconnect {
			ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "disconnecting"))
			break
		}

		var action = PlayerAction{
			playerId:   playerId,
			actionType: actionType,

			index: actionRequest.Index,
		}
		action.playerId = playerId
		lobby.ActionQueue <- action
	}

	quitAction := PlayerAction{
		playerId:   playerId,
		actionType: Disconnect,
	}
	lobby.ActionQueue <- quitAction
}

func main() {
	lobby := NewLobby()
	lobbies["test"] = lobby
	go lobby.run()

	http.HandleFunc("/lobby", handleCreateLobby)
	http.HandleFunc("/ws", handleWebSocket)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
