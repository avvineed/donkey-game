package game

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

var (
	ErrRoomNotFound       = errors.New("room not found")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrNotHost            = errors.New("only host can perform this action")
	ErrRoomFull           = errors.New("room is full")
	ErrInvalidPlayerCount = errors.New("player count must be between 3 and 13")
	ErrPlayerNotReady     = errors.New("all players must be ready")
	ErrGameAlreadyStarted = errors.New("game already started")
	ErrNotPlayersTurn     = errors.New("it is not your turn")
	ErrCardNotInHand      = errors.New("card not in hand")
	ErrMustPlayAceSpades  = errors.New("opening move must be the ace of spades")
	ErrMustFollowSuit     = errors.New("must follow active suit")
	ErrIllegalStrike      = errors.New("illegal strike: player had active suit available")
	ErrGameNotStarted     = errors.New("game not started")
	ErrSessionInvalid     = errors.New("invalid session")
	ErrPlayerNotConnected = errors.New("player not connected")
	ErrReconnectExpired   = errors.New("reconnect window has expired")
)

const reconnectGracePeriod = 3 * time.Minute

type Manager struct {
	mu    sync.RWMutex
	rng   *rand.Rand
	rooms map[string]*Room
}

func NewManager(seed int64) *Manager {
	return &Manager{
		rng:   rand.New(rand.NewSource(seed)),
		rooms: make(map[string]*Room),
	}
}

func (m *Manager) CreateRoom(nickname string) (*Room, *Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	roomID := newID("room")
	playerID := newID("player")
	sessionToken := newID("session")

	player := &Player{
		ID:           playerID,
		SessionToken: sessionToken,
		Nickname:     nickname,
		JoinOrder:    0,
		SeatIndex:    0,
		IsHost:       true,
		Ready:        false,
		Connected:    true,
	}

	room := &Room{
		ID:           roomID,
		Status:       RoomLobby,
		HostPlayerID: playerID,
		MinPlayers:   3,
		MaxPlayers:   13,
		CreatedAt:    time.Now().UTC(),
		Players:      []*Player{player},
		PlayerHands:  make(map[string][]Card),
		Sessions:     map[string]string{sessionToken: playerID},
		Connections:  make(map[string]Notifier),
	}
	m.rooms[roomID] = room
	return cloneRoom(room), clonePlayer(player), nil
}

func (m *Manager) JoinRoom(roomID, nickname string) (*Room, *Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return nil, nil, ErrRoomNotFound
	}
	if reclaimed := reclaimDisconnectedPlayer(room, nickname); reclaimed != nil {
		return cloneRoom(room), clonePlayer(reclaimed), nil
	}
	if room.Status != RoomLobby {
		return nil, nil, ErrGameAlreadyStarted
	}
	if len(room.Players) >= room.MaxPlayers {
		return nil, nil, ErrRoomFull
	}

	playerID := newID("player")
	sessionToken := newID("session")
	player := &Player{
		ID:           playerID,
		SessionToken: sessionToken,
		Nickname:     nickname,
		JoinOrder:    len(room.Players),
		SeatIndex:    len(room.Players),
		Connected:    true,
	}
	room.Players = append(room.Players, player)
	room.Sessions[sessionToken] = playerID
	return cloneRoom(room), clonePlayer(player), nil
}

func (m *Manager) Reconnect(roomID, sessionToken string) (*Room, *Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return nil, nil, ErrRoomNotFound
	}
	m.expirePausedGameIfNeeded(room)
	playerID, ok := room.Sessions[sessionToken]
	if !ok {
		return nil, nil, ErrSessionInvalid
	}
	player := getPlayer(room, playerID)
	if player == nil {
		return nil, nil, ErrPlayerNotFound
	}
	if room.Game != nil && room.Game.Stall != nil && time.Now().UTC().After(room.Game.Stall.ReconnectDeadline) {
		return nil, nil, ErrReconnectExpired
	}
	player.Connected = true
	delete(room.Connections, player.ID)
	m.resumeGameIfRecovered(room)
	return cloneRoom(room), clonePlayer(player), nil
}

func (m *Manager) SetReady(roomID, playerID string, ready bool) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return nil, ErrRoomNotFound
	}
	m.expirePausedGameIfNeeded(room)
	player := getPlayer(room, playerID)
	if player == nil {
		return nil, ErrPlayerNotFound
	}
	player.Ready = ready
	return cloneRoom(room), nil
}

func (m *Manager) StartGame(roomID, playerID string) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return nil, ErrRoomNotFound
	}
	m.expirePausedGameIfNeeded(room)
	if room.HostPlayerID != playerID {
		return nil, ErrNotHost
	}
	if room.Status != RoomLobby {
		return nil, ErrGameAlreadyStarted
	}
	if len(room.Players) < room.MinPlayers || len(room.Players) > room.MaxPlayers {
		return nil, ErrInvalidPlayerCount
	}
	for _, player := range room.Players {
		if !player.Ready {
			return nil, ErrPlayerNotReady
		}
	}

	deck := newDeck()
	m.rng.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	room.PlayerHands = dealHands(deck, room.Players)
	aceHolder := ""
	for _, player := range room.Players {
		sortCards(room.PlayerHands[player.ID])
		player.CardsRemaining = len(room.PlayerHands[player.ID])
		player.Finished = false
		player.FinishedAt = time.Time{}
		if hasAceOfSpades(room.PlayerHands[player.ID]) {
			aceHolder = player.ID
		}
	}

	room.Status = RoomInGame
	room.Game = &GameState{
		ID:            newID("game"),
		Phase:         PhaseAwaitingTurn,
		CurrentTurnID: aceHolder,
		LeadPlayerID:  aceHolder,
		StartedAt:     time.Now().UTC(),
		FinishedOrder: []string{},
		RecentActions: []RecentAction{},
		Round: RoundState{
			LeadPlayerID: aceHolder,
			TableCards:   []TableCard{},
		},
		PlayerHands: make(map[string]int),
		LastEvent:   "game_started",
	}
	for _, player := range room.Players {
		room.Game.PlayerHands[player.ID] = len(room.PlayerHands[player.ID])
	}
	return cloneRoom(room), nil
}

func (m *Manager) PlayCard(roomID, playerID string, card Card) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return nil, ErrRoomNotFound
	}
	m.expirePausedGameIfNeeded(room)
	if room.Game == nil || room.Status != RoomInGame {
		return nil, ErrGameNotStarted
	}
	if room.Game.Phase != PhaseAwaitingTurn {
		return nil, ErrGameNotStarted
	}
	if room.Game.CurrentTurnID != playerID {
		return nil, ErrNotPlayersTurn
	}

	hand := room.PlayerHands[playerID]
	cardIndex := indexOfCard(hand, card)
	if cardIndex < 0 {
		return nil, ErrCardNotInHand
	}
	if len(room.Game.Round.TableCards) == 0 {
		if room.Game.LeadPlayerID == playerID && room.Game.Round.DiscardCount == 0 && !card.Equal(Card{Suit: Spades, Rank: 14}) {
			return nil, ErrMustPlayAceSpades
		}
		room.Game.Round.ActiveSuit = card.Suit
	} else {
		activeSuit := room.Game.Round.ActiveSuit
		if card.Suit != activeSuit && handHasSuit(hand, activeSuit) {
			room.Status = RoomFinished
			room.Game.Phase = PhaseGameOver
			room.Game.LoserPlayerID = playerID
			room.Game.LastEvent = "illegal_strike"
			return cloneRoom(room), ErrIllegalStrike
		}
	}

	room.PlayerHands[playerID] = removeCardByIndex(hand, cardIndex)
	room.Game.Round.TableCards = append(room.Game.Round.TableCards, TableCard{PlayerID: playerID, Card: card})
	room.Game.PlayerHands[playerID] = len(room.PlayerHands[playerID])
	player := getPlayer(room, playerID)
	player.CardsRemaining = len(room.PlayerHands[playerID])
	appendRecentAction(room.Game, RecentAction{
		Type:      "play",
		PlayerID:  player.ID,
		Player:    player.Nickname,
		CardLabel: formatCardLabel(card),
		Message:   fmt.Sprintf("%s played %s.", player.Nickname, formatCardLabel(card)),
	})

	if len(room.Game.Round.TableCards) == 1 {
		room.Game.Round.LeadPlayerID = playerID
	}

	if isStrike(room.Game.Round.TableCards, room.Game.Round.ActiveSuit) {
		m.resolveStrike(room)
		return cloneRoom(room), nil
	}

	next := nextActivePlayer(room, playerID)
	if next == room.Game.Round.LeadPlayerID {
		m.resolveFullRound(room)
		return cloneRoom(room), nil
	}
	room.Game.CurrentTurnID = next
	room.Game.LastEvent = "card_played"
	return cloneRoom(room), nil
}

func (m *Manager) MarkDisconnected(roomID, playerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room, ok := m.rooms[roomID]
	if !ok {
		return
	}
	player := getPlayer(room, playerID)
	if player != nil {
		player.Connected = false
	}
	delete(room.Connections, playerID)
	if room.Game != nil && room.Status == RoomInGame && room.Game.Phase != PhaseGameOver {
		room.Game.Phase = PhasePaused
		room.Game.LastEvent = "player_disconnected"
		room.Game.Stall = &StallState{
			DisconnectedPlayerID: playerID,
			DisconnectedNickname: player.Nickname,
			Message:              fmt.Sprintf("%s has left the room. The game is stalled while they reconnect.", player.Nickname),
			ReconnectDeadline:    time.Now().UTC().Add(reconnectGracePeriod),
		}
	}
}

func (m *Manager) RegisterConnection(roomID, playerID string, notifier Notifier) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return ErrRoomNotFound
	}
	m.expirePausedGameIfNeeded(room)
	if getPlayer(room, playerID) == nil {
		return ErrPlayerNotFound
	}
	if existing := room.Connections[playerID]; existing != nil {
		_ = existing.Close()
	}
	room.Connections[playerID] = notifier
	return nil
}

func (m *Manager) BroadcastRoom(roomID string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return
	}
	m.expirePausedGameIfNeeded(room)
	for _, player := range room.Players {
		notifier := room.Connections[player.ID]
		if notifier == nil {
			continue
		}
		_ = notifier.Notify(EventEnvelope{
			Type: "state_sync",
			Data: publicStateForPlayer(room, player.ID),
		})
	}
}

func (m *Manager) PublicState(roomID, playerID string) (*PublicState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	room, ok := m.rooms[roomID]
	if !ok {
		return nil, ErrRoomNotFound
	}
	m.expirePausedGameIfNeeded(room)
	if getPlayer(room, playerID) == nil {
		return nil, ErrPlayerNotFound
	}
	state := publicStateForPlayer(room, playerID)
	return &state, nil
}

func (m *Manager) resolveStrike(room *Room) {
	table := room.Game.Round.TableCards
	activeSuit := room.Game.Round.ActiveSuit
	highestIdx := -1
	highestRank := Rank(0)
	for i, entry := range table {
		if entry.Card.Suit == activeSuit && entry.Card.Rank > highestRank {
			highestRank = entry.Card.Rank
			highestIdx = i
		}
	}
	if highestIdx < 0 {
		panic("strike resolution without active-suit card")
	}
	collectorID := table[highestIdx].PlayerID
	collected := make([]Card, 0, len(table))
	for _, entry := range table {
		if entry.Card.Suit == activeSuit || entry.Card.Suit != activeSuit {
			collected = append(collected, entry.Card)
		}
	}
	room.PlayerHands[collectorID] = append(room.PlayerHands[collectorID], collected...)
	sortCards(room.PlayerHands[collectorID])
	collector := getPlayer(room, collectorID)
	collector.CardsRemaining = len(room.PlayerHands[collectorID])
	room.Game.PlayerHands[collectorID] = collector.CardsRemaining
	room.Game.Round.DiscardCount += len(table) - len(collected)
	room.Game.Round.TableCards = []TableCard{}
	room.Game.Round.ActiveSuit = ""
	room.Game.LeadPlayerID = collectorID
	room.Game.CurrentTurnID = collectorID
	room.Game.LastEvent = "strike_occurred"
	appendRecentAction(room.Game, RecentAction{
		Type:      "cut",
		PlayerID:  table[len(table)-1].PlayerID,
		Player:    getPlayer(room, table[len(table)-1].PlayerID).Nickname,
		CardLabel: formatCardLabel(table[len(table)-1].Card),
		TargetID:  collectorID,
		Target:    collector.Nickname,
		Message: fmt.Sprintf(
			"%s cut with %s. %s collects the cards.",
			getPlayer(room, table[len(table)-1].PlayerID).Nickname,
			formatCardLabel(table[len(table)-1].Card),
			collector.Nickname,
		),
	})
	m.updateFinishedPlayers(room)
	m.finishIfNeeded(room)
}

func (m *Manager) resolveFullRound(room *Room) {
	activeSuit := room.Game.Round.ActiveSuit
	highest := TableCard{}
	for _, entry := range room.Game.Round.TableCards {
		if entry.Card.Suit == activeSuit && entry.Card.Rank > highest.Card.Rank {
			highest = entry
		}
	}
	room.Game.Round.DiscardCount += len(room.Game.Round.TableCards)
	room.Game.Round.TableCards = []TableCard{}
	room.Game.Round.ActiveSuit = ""
	room.Game.LeadPlayerID = highest.PlayerID
	room.Game.CurrentTurnID = highest.PlayerID
	room.Game.LastEvent = "round_resolved"
	appendRecentAction(room.Game, RecentAction{
		Type:      "round_win",
		PlayerID:  highest.PlayerID,
		Player:    getPlayer(room, highest.PlayerID).Nickname,
		CardLabel: formatCardLabel(highest.Card),
		Message:   fmt.Sprintf("%s won the round with %s.", getPlayer(room, highest.PlayerID).Nickname, formatCardLabel(highest.Card)),
	})
	m.updateFinishedPlayers(room)
	m.finishIfNeeded(room)
}

func (m *Manager) updateFinishedPlayers(room *Room) {
	for _, player := range room.Players {
		if !player.Finished && len(room.PlayerHands[player.ID]) == 0 {
			player.Finished = true
			player.FinishedAt = time.Now().UTC()
			room.Game.FinishedOrder = append(room.Game.FinishedOrder, player.ID)
		}
		player.CardsRemaining = len(room.PlayerHands[player.ID])
		room.Game.PlayerHands[player.ID] = player.CardsRemaining
	}
}

func (m *Manager) finishIfNeeded(room *Room) {
	active := activePlayers(room)
	if len(active) > 1 {
		room.Game.Phase = PhaseAwaitingTurn
		room.Game.Stall = nil
		return
	}
	if len(active) == 1 {
		room.Game.LoserPlayerID = active[0].ID
	}
	if len(room.Game.FinishedOrder) > 0 {
		room.Game.WinnerPlayerID = room.Game.FinishedOrder[0]
	}
	room.Game.Phase = PhaseGameOver
	room.Status = RoomFinished
	room.Game.LastEvent = "game_ended"
	room.Game.Stall = nil
}

func (m *Manager) resumeGameIfRecovered(room *Room) {
	if room.Game == nil || room.Game.Stall == nil {
		return
	}
	for _, player := range room.Players {
		if !player.Connected {
			return
		}
	}
	room.Game.Phase = PhaseAwaitingTurn
	room.Game.LastEvent = "player_reconnected"
	room.Game.Stall = nil
}

func (m *Manager) expirePausedGameIfNeeded(room *Room) {
	if room.Game == nil || room.Game.Stall == nil {
		return
	}
	if time.Now().UTC().Before(room.Game.Stall.ReconnectDeadline) {
		return
	}
	room.Game.Phase = PhaseGameOver
	room.Status = RoomFinished
	room.Game.LastEvent = "reconnect_expired"
	room.Game.Stall.Message = fmt.Sprintf("%s did not return in time. The stalled game has ended.", room.Game.Stall.DisconnectedNickname)
}

func reclaimDisconnectedPlayer(room *Room, nickname string) *Player {
	if room.Game == nil || room.Game.Stall == nil {
		return nil
	}
	if time.Now().UTC().After(room.Game.Stall.ReconnectDeadline) {
		return nil
	}
	for _, player := range room.Players {
		if player.Nickname == nickname && !player.Connected {
			oldToken := player.SessionToken
			if oldToken != "" {
				delete(room.Sessions, oldToken)
			}
			player.SessionToken = newID("session")
			player.Connected = true
			room.Sessions[player.SessionToken] = player.ID
			return player
		}
	}
	return nil
}

func appendRecentAction(game *GameState, action RecentAction) {
	game.RecentActions = append([]RecentAction{action}, game.RecentActions...)
	if len(game.RecentActions) > 5 {
		game.RecentActions = game.RecentActions[:5]
	}
}

func formatCardLabel(card Card) string {
	ranks := map[Rank]string{
		11: "J",
		12: "Q",
		13: "K",
		14: "A",
	}
	suits := map[Suit]string{
		Spades:   "Spades",
		Hearts:   "Hearts",
		Diamonds: "Diamonds",
		Clubs:    "Clubs",
	}
	rank := ranks[card.Rank]
	if rank == "" {
		rank = itoa(int(card.Rank))
	}
	return fmt.Sprintf("%s of %s", rank, suits[card.Suit])
}

func publicStateForPlayer(room *Room, playerID string) PublicState {
	publicPlayers := make([]Player, 0, len(room.Players))
	readyCount := 0
	for _, player := range room.Players {
		if player.Ready {
			readyCount++
		}
		publicPlayers = append(publicPlayers, *clonePlayer(player))
	}
	publicRoom := &PublicRoom{
		ID:           room.ID,
		Status:       room.Status,
		HostPlayerID: room.HostPlayerID,
		MinPlayers:   room.MinPlayers,
		MaxPlayers:   room.MaxPlayers,
		CreatedAt:    room.CreatedAt,
		Players:      publicPlayers,
		ReadyCount:   readyCount,
		PlayerCount:  len(publicPlayers),
	}
	if room.Game != nil {
		gameCopy := *room.Game
		gameCopy.PlayerHands = nil
		if gameCopy.FinishedOrder == nil {
			gameCopy.FinishedOrder = []string{}
		}
		if gameCopy.Round.TableCards == nil {
			gameCopy.Round.TableCards = []TableCard{}
		}
		publicRoom.Game = &gameCopy
	}
	state := PublicState{Room: publicRoom}
	if cards, ok := room.PlayerHands[playerID]; ok {
		hand := append([]Card(nil), cards...)
		sortCards(hand)
		state.PrivateHand = &PlayerPrivateState{PlayerID: playerID, Hand: hand}
	}
	return state
}

func getPlayer(room *Room, playerID string) *Player {
	for _, player := range room.Players {
		if player.ID == playerID {
			return player
		}
	}
	return nil
}

func cloneRoom(room *Room) *Room {
	copied := *room
	copied.Players = make([]*Player, 0, len(room.Players))
	for _, player := range room.Players {
		copied.Players = append(copied.Players, clonePlayer(player))
	}
	copied.PlayerHands = make(map[string][]Card, len(room.PlayerHands))
	for id, hand := range room.PlayerHands {
		copied.PlayerHands[id] = append([]Card(nil), hand...)
	}
	if room.Game != nil {
		gameCopy := *room.Game
		gameCopy.Round.TableCards = append([]TableCard(nil), room.Game.Round.TableCards...)
		gameCopy.PlayerHands = make(map[string]int, len(room.Game.PlayerHands))
		for id, count := range room.Game.PlayerHands {
			gameCopy.PlayerHands[id] = count
		}
		gameCopy.FinishedOrder = append([]string(nil), room.Game.FinishedOrder...)
		copied.Game = &gameCopy
	}
	return &copied
}

func clonePlayer(player *Player) *Player {
	copied := *player
	return &copied
}

func newDeck() []Card {
	suits := []Suit{Spades, Hearts, Diamonds, Clubs}
	deck := make([]Card, 0, 52)
	for _, suit := range suits {
		for rank := 2; rank <= 14; rank++ {
			deck = append(deck, Card{Suit: suit, Rank: Rank(rank)})
		}
	}
	return deck
}

func dealHands(deck []Card, players []*Player) map[string][]Card {
	hands := make(map[string][]Card, len(players))
	base := len(deck) / len(players)
	remainder := len(deck) % len(players)
	offset := 0
	for i, player := range players {
		count := base
		if i < remainder {
			count++
		}
		hands[player.ID] = append([]Card(nil), deck[offset:offset+count]...)
		offset += count
	}
	return hands
}

func sortCards(cards []Card) {
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Suit == cards[j].Suit {
			return cards[i].Rank < cards[j].Rank
		}
		return suitOrder(cards[i].Suit) < suitOrder(cards[j].Suit)
	})
}

func suitOrder(s Suit) int {
	switch s {
	case Spades:
		return 0
	case Hearts:
		return 1
	case Diamonds:
		return 2
	default:
		return 3
	}
}

func hasAceOfSpades(hand []Card) bool {
	return indexOfCard(hand, Card{Suit: Spades, Rank: 14}) >= 0
}

func handHasSuit(hand []Card, suit Suit) bool {
	for _, card := range hand {
		if card.Suit == suit {
			return true
		}
	}
	return false
}

func indexOfCard(hand []Card, target Card) int {
	for i, card := range hand {
		if card.Equal(target) {
			return i
		}
	}
	return -1
}

func removeCardByIndex(hand []Card, idx int) []Card {
	next := append([]Card(nil), hand[:idx]...)
	next = append(next, hand[idx+1:]...)
	return next
}

func nextActivePlayer(room *Room, currentPlayerID string) string {
	if len(room.Players) == 0 {
		return ""
	}
	currentSeat := getPlayer(room, currentPlayerID).SeatIndex
	for i := 1; i <= len(room.Players); i++ {
		candidate := room.Players[(currentSeat+i)%len(room.Players)]
		if !candidate.Finished {
			return candidate.ID
		}
	}
	return currentPlayerID
}

func activePlayers(room *Room) []*Player {
	players := make([]*Player, 0, len(room.Players))
	for _, player := range room.Players {
		if !player.Finished {
			players = append(players, player)
		}
	}
	return players
}

func isStrike(table []TableCard, activeSuit Suit) bool {
	if len(table) < 2 {
		return false
	}
	last := table[len(table)-1]
	return last.Card.Suit != activeSuit
}

func ValidateNickname(name string) error {
	if len(name) < 2 || len(name) > 24 {
		return fmt.Errorf("nickname must be between 2 and 24 characters")
	}
	return nil
}
