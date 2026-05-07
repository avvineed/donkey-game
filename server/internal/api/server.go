package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/vineed/games/kazhuta/server/internal/game"
)

type Server struct {
	manager  *game.Manager
	upgrader websocket.Upgrader
}

func NewServer(manager *game.Manager) *Server {
	return &Server{
		manager: manager,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/rooms", s.handleCreateRoom)
	mux.HandleFunc("/api/rooms/join", s.handleJoinRoom)
	mux.HandleFunc("/api/rooms/reconnect", s.handleReconnect)
	mux.HandleFunc("/api/rooms/ready", s.handleSetReady)
	mux.HandleFunc("/api/rooms/start", s.handleStartGame)
	mux.HandleFunc("/api/games/play", s.handlePlayCard)
	mux.HandleFunc("/ws", s.handleWebSocket)
	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type createRoomRequest struct {
	Nickname string `json:"nickname"`
}

type roomActionResponse struct {
	PlayerID     string            `json:"playerId"`
	SessionToken string            `json:"sessionToken"`
	State        *game.PublicState `json:"state"`
}

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req createRoomRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := game.ValidateNickname(req.Nickname); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	room, player, err := s.manager.CreateRoom(req.Nickname)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	state := mustPublicState(s.manager, room.ID, player.ID)
	writeJSON(w, http.StatusCreated, roomActionResponse{
		PlayerID:     player.ID,
		SessionToken: player.SessionToken,
		State:        state,
	})
}

type joinRoomRequest struct {
	RoomID   string `json:"roomId"`
	Nickname string `json:"nickname"`
}

func (s *Server) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req joinRoomRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := game.ValidateNickname(req.Nickname); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	room, player, err := s.manager.JoinRoom(req.RoomID, req.Nickname)
	if err != nil {
		writeGameError(w, err)
		return
	}
	s.manager.BroadcastRoom(room.ID)
	state := mustPublicState(s.manager, room.ID, player.ID)
	writeJSON(w, http.StatusOK, roomActionResponse{
		PlayerID:     player.ID,
		SessionToken: player.SessionToken,
		State:        state,
	})
}

type reconnectRequest struct {
	RoomID       string `json:"roomId"`
	SessionToken string `json:"sessionToken"`
}

func (s *Server) handleReconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req reconnectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	room, player, err := s.manager.Reconnect(req.RoomID, req.SessionToken)
	if err != nil {
		writeGameError(w, err)
		return
	}
	s.manager.BroadcastRoom(room.ID)
	state := mustPublicState(s.manager, room.ID, player.ID)
	writeJSON(w, http.StatusOK, roomActionResponse{
		PlayerID:     player.ID,
		SessionToken: player.SessionToken,
		State:        state,
	})
}

type readyRequest struct {
	RoomID   string `json:"roomId"`
	PlayerID string `json:"playerId"`
	Ready    bool   `json:"ready"`
}

func (s *Server) handleSetReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req readyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	room, err := s.manager.SetReady(req.RoomID, req.PlayerID, req.Ready)
	if err != nil {
		writeGameError(w, err)
		return
	}
	s.manager.BroadcastRoom(room.ID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type startRequest struct {
	RoomID   string `json:"roomId"`
	PlayerID string `json:"playerId"`
}

func (s *Server) handleStartGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req startRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	room, err := s.manager.StartGame(req.RoomID, req.PlayerID)
	if err != nil {
		writeGameError(w, err)
		return
	}
	s.manager.BroadcastRoom(room.ID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type playCardRequest struct {
	RoomID   string    `json:"roomId"`
	PlayerID string    `json:"playerId"`
	Card     game.Card `json:"card"`
}

func (s *Server) handlePlayCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req playCardRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	room, err := s.manager.PlayCard(req.RoomID, req.PlayerID, req.Card)
	if err != nil && !errors.Is(err, game.ErrIllegalStrike) {
		writeGameError(w, err)
		return
	}
	s.manager.BroadcastRoom(room.ID)
	if errors.Is(err, game.ErrIllegalStrike) {
		writeGameError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("roomId")
	playerID := r.URL.Query().Get("playerId")
	if roomID == "" || playerID == "" {
		writeError(w, http.StatusBadRequest, "roomId and playerId are required")
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	notifier := &wsNotifier{conn: conn}
	if err := s.manager.RegisterConnection(roomID, playerID, notifier); err != nil {
		_ = conn.Close()
		return
	}
	if state, err := s.manager.PublicState(roomID, playerID); err == nil {
		_ = notifier.Notify(game.EventEnvelope{Type: "state_sync", Data: state})
	}
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			s.manager.MarkDisconnected(roomID, playerID)
			s.manager.BroadcastRoom(roomID)
			_ = conn.Close()
			return
		}
	}
}

type wsNotifier struct {
	conn *websocket.Conn
}

func (w *wsNotifier) Notify(event game.EventEnvelope) error {
	return w.conn.WriteJSON(event)
}

func (w *wsNotifier) Close() error {
	return w.conn.Close()
}

func mustPublicState(manager *game.Manager, roomID, playerID string) *game.PublicState {
	state, _ := manager.PublicState(roomID, playerID)
	return state
}

func decodeJSON(r *http.Request, out interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeGameError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, game.ErrRoomNotFound), errors.Is(err, game.ErrPlayerNotFound):
		status = http.StatusNotFound
	case errors.Is(err, game.ErrNotHost):
		status = http.StatusForbidden
	case errors.Is(err, game.ErrGameAlreadyStarted):
		status = http.StatusConflict
	case errors.Is(err, game.ErrReconnectExpired):
		status = http.StatusGone
	}
	writeError(w, status, err.Error())
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
