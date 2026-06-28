package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

const (
	// pings exist so cloudflare doesn't close connections
	pongWait   = 60 * time.Second
	pingPeriod = 45 * time.Second
	writeWait  = 10 * time.Second
)

type CreateLobbyRequest struct {
	Name string `json:"name"`
}

type ActionRequest struct {
	ActionStr string `json:"action"`

	// optional fields
	PlaceKittenIndex int    `json:"placeKittenIndex"` // for placing kittens
	UseCardIndex     int    `json:"useCardIndex"`     // card that you place
	AlterFutureOrder []int  `json:"alterFutureOrder"` // new order of first 3 cards (e.g., [2, 1, 0] to reverse)
	TargetedPlayer   int    `json:"targetedPlayer"`   // player being targeted
	ComboIndices     []int  `json:"comboIndices"`     // list of cards used for combo
	RequestedCardStr string `json:"requestedCard"`    // card requested for combo
	WantNoped        bool   `json:"wantNoped"`        // for PLAY_NOPE: true = nope, false = yup
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

	// Discord auto-join passes create=1 so the instance lobby is created on the
	// fly; the website "join" flow omits it and still 404s on a missing lobby.
	create := r.URL.Query().Get("create") == "1"

	// find if lobby exists, creating it atomically when auto-create is requested
	// so simultaneous joiners can't double-create the same lobby
	lobbiesMutex.Lock()
	lobby, exists := lobbies[lobbyName]
	if !exists {
		if !create {
			lobbiesMutex.Unlock()
			http.Error(w, "Lobby not found. Create it first.", http.StatusNotFound)
			return
		}
		lobby = NewLobby()
		lobbies[lobbyName] = lobby
		go lobby.run()
		log.Printf("Lobby auto-created: %s", lobbyName)
	}
	lobbiesMutex.Unlock()

	// upgrade the connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

	// request to join lobby
	username := r.URL.Query().Get("username")
	userId := r.URL.Query().Get("userId")
	avatar := r.URL.Query().Get("avatar")
	// Buffered so a briefly-slow client doesn't block the lobby goroutine on
	// send; a genuinely wedged client fills the buffer and is dropped (see
	// sendTo). Sized for many pending states without growing unbounded.
	gameStateChan := make(chan GameState, 16)
	joinResultChan := make(chan JoinResponse)
	joinReq := JoinRequest{
		Name:   username,
		UserId: userId,
		Avatar: avatar,
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

	// send messages and pings to client. pings prevent auto socket disconnect.
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case state, ok := <-gameStateChan:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					// shut socket down if player is disconnected
					ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					ws.Close()
					return
				}
				if err := ws.WriteJSON(state); err != nil {
					log.Println("Write error:", err)
					ws.Close()
					return
				}
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					ws.Close()
					return
				}
			}
		}
	}()

	// disconnect client who doesn't respond to ping
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Spectators are watch-only: they receive broadcasts via the writer goroutine
	// above but never drive game actions. We only read to detect their departure.
	if joinResponse.isSpectator {
		log.Printf("Spectator %d joined lobby %s", playerId, lobbyName)
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}
		lobby.ActionQueue <- PlayerAction{
			playerId:    playerId,
			actionType:  Disconnect,
			isSpectator: true,
			conn:        gameStateChan,
		}
		return
	}

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

			placeKittenIndex: actionRequest.PlaceKittenIndex,
			useCardIndex:     actionRequest.UseCardIndex,
			alterFutureOrder: actionRequest.AlterFutureOrder,
			targetedPlayer:   actionRequest.TargetedPlayer,
			comboIndices:     actionRequest.ComboIndices,
			wantNoped:        actionRequest.WantNoped,
		}

		if actionRequest.RequestedCardStr != "" {
			requestedCard, err := ParseCard(actionRequest.RequestedCardStr)
			if err != nil {
				lobby.sendError(playerId, err.Error())
				continue
			}
			action.requestedCard = requestedCard
		}

		lobby.ActionQueue <- action
	}

	quitAction := PlayerAction{
		playerId:   playerId,
		actionType: Disconnect,
		conn:       gameStateChan,
	}
	lobby.ActionQueue <- quitAction
}

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("No .env file loaded: %v", err)
	}

	lobby := NewLobby()
	lobbies["test"] = lobby
	go lobby.run()

	// Routes are served under /api so a single path prefix works across every
	// environment: the Vite dev proxy, the Vercel rewrite, and the Discord
	// activity URL mapping all forward /api verbatim (none of them strip it).
	http.HandleFunc("/api/lobby", handleCreateLobby)
	http.HandleFunc("/api/ws", handleWebSocket)
	http.HandleFunc("/api/token", handleToken)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
