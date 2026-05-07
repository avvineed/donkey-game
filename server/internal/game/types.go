package game

import "time"

type Suit string

const (
	Spades   Suit = "spades"
	Hearts   Suit = "hearts"
	Diamonds Suit = "diamonds"
	Clubs    Suit = "clubs"
)

type Rank int

type Card struct {
	Suit Suit `json:"suit"`
	Rank Rank `json:"rank"`
}

func (c Card) Equal(other Card) bool {
	return c.Suit == other.Suit && c.Rank == other.Rank
}

func (c Card) ID() string {
	return string(c.Suit) + "-" + itoa(int(c.Rank))
}

type RoomStatus string

const (
	RoomLobby    RoomStatus = "lobby"
	RoomInGame   RoomStatus = "in_game"
	RoomFinished RoomStatus = "finished"
)

type GamePhase string

const (
	PhaseLobby        GamePhase = "lobby"
	PhaseAwaitingTurn GamePhase = "awaiting_turn"
	PhaseRoundEnd     GamePhase = "round_end"
	PhaseGameOver     GamePhase = "game_over"
	PhasePaused       GamePhase = "paused"
)

type TableCard struct {
	PlayerID string `json:"playerId"`
	Card     Card   `json:"card"`
}

type Player struct {
	ID             string    `json:"playerId"`
	SessionToken   string    `json:"-"`
	Nickname       string    `json:"nickname"`
	JoinOrder      int       `json:"joinOrder"`
	SeatIndex      int       `json:"seatIndex"`
	IsHost         bool      `json:"isHost"`
	Ready          bool      `json:"ready"`
	Connected      bool      `json:"connected"`
	CardsRemaining int       `json:"cardsRemaining"`
	Finished       bool      `json:"finished"`
	FinishedAt     time.Time `json:"-"`
}

type PlayerPrivateState struct {
	PlayerID string `json:"playerId"`
	Hand     []Card `json:"hand"`
}

type RoundState struct {
	LeadPlayerID string      `json:"leadPlayerId"`
	ActiveSuit   Suit        `json:"activeSuit"`
	TableCards   []TableCard `json:"tableCards"`
	DiscardCount int         `json:"discardCount"`
}

type GameState struct {
	ID             string         `json:"gameId"`
	Phase          GamePhase      `json:"phase"`
	CurrentTurnID  string         `json:"currentTurnPlayerId"`
	LeadPlayerID   string         `json:"leadPlayerId"`
	StartedAt      time.Time      `json:"startedAt"`
	FinishedOrder  []string       `json:"finishedOrder"`
	LoserPlayerID  string         `json:"loserPlayerId"`
	WinnerPlayerID string         `json:"winnerPlayerId"`
	Round          RoundState     `json:"round"`
	LastEvent      string         `json:"lastEvent"`
	Stall          *StallState    `json:"stall,omitempty"`
	PlayerHands    map[string]int `json:"-"`
}

type StallState struct {
	DisconnectedPlayerID string    `json:"disconnectedPlayerId"`
	DisconnectedNickname string    `json:"disconnectedNickname"`
	Message              string    `json:"message"`
	ReconnectDeadline    time.Time `json:"reconnectDeadline"`
}

type Room struct {
	ID           string              `json:"roomId"`
	Status       RoomStatus          `json:"status"`
	HostPlayerID string              `json:"hostPlayerId"`
	MinPlayers   int                 `json:"minPlayers"`
	MaxPlayers   int                 `json:"maxPlayers"`
	CreatedAt    time.Time           `json:"createdAt"`
	Players      []*Player           `json:"players"`
	PlayerHands  map[string][]Card   `json:"-"`
	Game         *GameState          `json:"game,omitempty"`
	Sessions     map[string]string   `json:"-"`
	Connections  map[string]Notifier `json:"-"`
}

type PublicState struct {
	Room        *PublicRoom         `json:"room"`
	PrivateHand *PlayerPrivateState `json:"privateHand,omitempty"`
}

type PublicRoom struct {
	ID           string     `json:"roomId"`
	Status       RoomStatus `json:"status"`
	HostPlayerID string     `json:"hostPlayerId"`
	MinPlayers   int        `json:"minPlayers"`
	MaxPlayers   int        `json:"maxPlayers"`
	CreatedAt    time.Time  `json:"createdAt"`
	Players      []Player   `json:"players"`
	Game         *GameState `json:"game,omitempty"`
	ReadyCount   int        `json:"readyCount"`
	PlayerCount  int        `json:"playerCount"`
}

type EventEnvelope struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Notifier interface {
	Notify(EventEnvelope) error
	Close() error
}
