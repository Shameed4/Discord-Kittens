package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
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
const proxyHeader = "X-Kittens-Proxied"

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

func isOwnAddress(addr string) bool {
	return addr == cfg.AdvertiseAddr
}

// serves as a proxy connection between client and host node
func proxyWebSocket(client *websocket.Conn, ownerAddr string, r *http.Request) {
	target := url.URL{Scheme: "ws", Host: ownerAddr, Path: "/api/ws", RawQuery: r.URL.RawQuery}

	header := http.Header{}
	header.Set(proxyHeader, "1") // loop guard

	upstream, resp, err := websocket.DefaultDialer.Dial(target.String(), header)
	if err != nil {
		log.Printf("proxy dial to %s failed: %v", ownerAddr, err)
		client.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(4000, "Could not reach lobby host"))
		return
	}
	if resp != nil {
		resp.Body.Close()
	}
	defer upstream.Close()

	// The proxy is transparent: it forwards ping/pong across the hop rather than
	// answering them, so the owner node's keepalive/liveness heartbeat reaches the
	// real client end-to-end (and a dead client is detected there, not masked here).
	forwardControl(client, upstream)

	errc := make(chan error, 2)
	go pumpWS(upstream, client, errc)
	go pumpWS(client, upstream, errc)
	<-errc
}

// relays all pings and pongs between the client and the upstream
func forwardControl(client, upstream *websocket.Conn) {
	client.SetPingHandler(func(data string) error {
		return upstream.WriteControl(websocket.PingMessage, []byte(data), time.Now().Add(writeWait))
	})
	upstream.SetPingHandler(func(data string) error {
		return client.WriteControl(websocket.PingMessage, []byte(data), time.Now().Add(writeWait))
	})
	client.SetPongHandler(func(data string) error {
		return upstream.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(writeWait))
	})
	upstream.SetPongHandler(func(data string) error {
		return client.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(writeWait))
	})
}

// relays messages from src to dst, writing to error channel if an error occurs.
func pumpWS(dst, src *websocket.Conn, errc chan error) {
	for {
		mt, data, err := src.ReadMessage()
		if err != nil {
			errc <- err
			return
		}
		dst.SetWriteDeadline(time.Now().Add(writeWait))
		if err := dst.WriteMessage(mt, data); err != nil {
			errc <- err
			return
		}
	}
}

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
	_, exists := lobbies[req.Name]
	lobbiesMutex.Unlock()

	// Check if lobby already exists
	if exists {
		http.Error(w, "Lobby already exists", http.StatusConflict)
		return
	}

	addr, epoch, err := coordinator.Acquire(req.Name)
	if err != nil {
		http.Error(w, "Internal error", http.StatusServiceUnavailable)
		return
	} else if !isOwnAddress(addr) {
		log.Printf("Tried to create lobby but it already exists in %s", addr)
		http.Error(w, "Lobby already exists", http.StatusConflict)
		return
	} else {
		lobbiesMutex.Lock()
		_, exists := lobbies[req.Name]
		if exists {
			lobbiesMutex.Unlock()
			http.Error(w, "Lobby already exists", http.StatusConflict)
			return
		} else {
			lobby := NewLobby(req.Name, epoch)
			lobbies[req.Name] = lobby
			go lobby.run()
		}
		lobbiesMutex.Unlock()
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status": "created"}`))
	log.Printf("Lobby created: %s", req.Name)
}

// resolveLobby returns either a live lobby, the node that contains the live lobby, or
// an error, depending on if the create flag is used and if the create flag is set.
// the caller must not hold the lobbies mutex.
func resolveLobby(name string, create bool) (*Lobby, string, error) {
	lobbiesMutex.Lock()
	lobby, existsLocally := lobbies[name]
	lobbiesMutex.Unlock()
	var ownerAddr string
	var epoch int64
	var err error
	if !existsLocally {
		if !create {
			ownerAddr, err = coordinator.Lookup(name)
		} else {
			ownerAddr, epoch, err = coordinator.Acquire(name)
			if isOwnAddress(ownerAddr) {
				lobbiesMutex.Lock()
				defer lobbiesMutex.Unlock()
				lobby, existsLocally = lobbies[name]
				if !existsLocally {
					lobby = NewLobby(name, epoch)
					lobbies[name] = lobby
					go lobby.run()
					log.Printf("Lobby auto-created: %s", name)
				}
			}
		}
	}
	return lobby, ownerAddr, err
}

func sendRejectedResolveLobby(lobby *Lobby, ownerAddr string, err error, ws *websocket.Conn) bool {
	if err != nil {
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4000, "Internal server error - please try again"))
		return true
	} else if lobby == nil && ownerAddr == "" {
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4000, "Lobby not found - create it first"))
		return true
	}
	return false
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Println("New request to join lobby")
	lobbyName := r.URL.Query().Get("lobby")
	if lobbyName == "" {
		http.Error(w, "Missing lobby parameter", http.StatusBadRequest)
		return
	}

	// Discord auto-join passes create=1 so the instance lobby is created on the
	// fly; the website "join" flow omits it, so a missing lobby is rejected below.
	create := r.URL.Query().Get("create") == "1"

	// Upgrade before resolving the lobby. A pre-upgrade HTTP 404 reaches the
	// browser only as a codeless 1006 close — the WebSocket API hides the status,
	// so the client can't distinguish a bad lobby name from a network blip and
	// retries forever. Upgrading first lets us reject with an application close
	// code (4000) + reason the client can read and act on, matching the
	// reaped-mid-join path below.
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

	var (
		lobby *Lobby
		addr  string
	)
	for {
		lobby, addr, err = resolveLobby(lobbyName, create)
		if sendRejectedResolveLobby(lobby, addr, err, ws) {
			return
		}

		if lobby == nil {
			if r.Header.Get(proxyHeader) != "" {
				ws.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(4000, "Lobby owner unavailable"))
				return
			}
			proxyWebSocket(ws, addr, r)
			return
		}

		// Hand the join to the lobby goroutine. The lobby may have been reaped for
		// inactivity between resolveLobby and now (notably during the WS upgrade);
		// done is closed on reap, so re-resolve and retry instead of blocking
		// forever on a dead goroutine's JoinQueue.
		select {
		case lobby.JoinQueue <- joinReq:
		case <-lobby.done:
			continue
		}
		break
	}

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

	if joinResponse.isSpectator {
		log.Printf("Spectator %d joined lobby %s", playerId, lobbyName)
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

	cfg = LoadConfig()
	coordinator = newCoordinator()
	// Routes are served under /api so a single path prefix works across every
	// environment: the Vite dev proxy, the Vercel rewrite, and the Discord
	// activity URL mapping all forward /api verbatim (none of them strip it).
	http.HandleFunc("/api/lobby", handleCreateLobby)
	http.HandleFunc("/api/ws", handleWebSocket)
	http.HandleFunc("/api/token", handleToken)

	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
