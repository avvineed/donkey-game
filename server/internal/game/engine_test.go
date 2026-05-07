package game

import (
	"testing"
	"time"
)

func TestDealHandsUsesAllCardsAndRemainderByJoinOrder(t *testing.T) {
	players := []*Player{
		{ID: "p1", JoinOrder: 0},
		{ID: "p2", JoinOrder: 1},
		{ID: "p3", JoinOrder: 2},
	}
	hands := dealHands(newDeck(), players)
	if got := len(hands["p1"]); got != 18 {
		t.Fatalf("p1 cards = %d, want 18", got)
	}
	if got := len(hands["p2"]); got != 17 {
		t.Fatalf("p2 cards = %d, want 17", got)
	}
	if got := len(hands["p3"]); got != 17 {
		t.Fatalf("p3 cards = %d, want 17", got)
	}
	total := 0
	for _, hand := range hands {
		total += len(hand)
	}
	if total != 52 {
		t.Fatalf("total cards = %d, want 52", total)
	}
}

func TestStartGameAssignsAceHolderFirstTurn(t *testing.T) {
	manager := NewManager(1)
	room, host, _ := manager.CreateRoom("host")
	_, p2, _ := manager.JoinRoom(room.ID, "p2")
	_, p3, _ := manager.JoinRoom(room.ID, "p3")
	manager.SetReady(room.ID, host.ID, true)
	manager.SetReady(room.ID, p2.ID, true)
	manager.SetReady(room.ID, p3.ID, true)

	started, err := manager.StartGame(room.ID, host.ID)
	if err != nil {
		t.Fatalf("StartGame error = %v", err)
	}
	if started.Game.CurrentTurnID == "" {
		t.Fatal("expected first turn holder")
	}
	found := false
	for playerID, hand := range started.PlayerHands {
		if hasAceOfSpades(hand) && playerID == started.Game.CurrentTurnID {
			found = true
		}
	}
	if !found {
		t.Fatal("ace of spades holder did not receive first turn")
	}
}

func TestPlayCardRejectsIllegalFollowSuitViolation(t *testing.T) {
	manager := NewManager(1)
	room := &Room{
		ID:           "room",
		Status:       RoomInGame,
		HostPlayerID: "p1",
		Players: []*Player{
			{ID: "p1", SeatIndex: 0},
			{ID: "p2", SeatIndex: 1},
			{ID: "p3", SeatIndex: 2},
		},
		PlayerHands: map[string][]Card{
			"p1": {{Suit: Spades, Rank: 14}},
			"p2": {{Suit: Spades, Rank: 10}, {Suit: Hearts, Rank: 2}},
			"p3": {{Suit: Clubs, Rank: 5}},
		},
		Game: &GameState{
			Phase:         PhaseAwaitingTurn,
			CurrentTurnID: "p2",
			LeadPlayerID:  "p1",
			Round: RoundState{
				LeadPlayerID: "p1",
				ActiveSuit:   Spades,
				TableCards:   []TableCard{{PlayerID: "p1", Card: Card{Suit: Spades, Rank: 14}}},
			},
			PlayerHands: map[string]int{"p1": 1, "p2": 2, "p3": 1},
		},
	}
	manager.rooms[room.ID] = room

	updated, err := manager.PlayCard(room.ID, "p2", Card{Suit: Hearts, Rank: 2})
	if err != ErrIllegalStrike {
		t.Fatalf("err = %v, want %v", err, ErrIllegalStrike)
	}
	if updated.Game.LoserPlayerID != "p2" {
		t.Fatalf("loser = %s, want p2", updated.Game.LoserPlayerID)
	}
}

func TestGameEndsWhenOnePlayerHasCardsLeft(t *testing.T) {
	manager := NewManager(1)
	room := &Room{
		ID: "room",
		Players: []*Player{
			{ID: "p1", SeatIndex: 0, Finished: true},
			{ID: "p2", SeatIndex: 1, Finished: true},
			{ID: "p3", SeatIndex: 2},
		},
		PlayerHands: map[string][]Card{
			"p1": {},
			"p2": {},
			"p3": {{Suit: Spades, Rank: 2}},
		},
		Game: &GameState{
			PlayerHands:   map[string]int{"p1": 0, "p2": 0, "p3": 1},
			FinishedOrder: []string{"p1", "p2"},
		},
	}
	manager.finishIfNeeded(room)
	if room.Game.Phase != PhaseGameOver {
		t.Fatalf("phase = %s, want %s", room.Game.Phase, PhaseGameOver)
	}
	if room.Game.LoserPlayerID != "p3" {
		t.Fatalf("loser = %s, want p3", room.Game.LoserPlayerID)
	}
}

func TestDisconnectPausesLiveGame(t *testing.T) {
	manager := NewManager(1)
	room := &Room{
		ID:     "room",
		Status: RoomInGame,
		Players: []*Player{
			{ID: "p1", Nickname: "A", Connected: true},
			{ID: "p2", Nickname: "B", Connected: true},
		},
		Game: &GameState{
			Phase:         PhaseAwaitingTurn,
			CurrentTurnID: "p2",
		},
		Connections: map[string]Notifier{},
	}
	manager.rooms[room.ID] = room

	manager.MarkDisconnected(room.ID, "p2")

	if room.Game.Phase != PhasePaused {
		t.Fatalf("phase = %s, want %s", room.Game.Phase, PhasePaused)
	}
	if room.Game.Stall == nil || room.Game.Stall.DisconnectedPlayerID != "p2" {
		t.Fatal("expected stall state for disconnected player")
	}
}

func TestJoinRoomReclaimsDisconnectedPlayerDuringStall(t *testing.T) {
	manager := NewManager(1)
	room := &Room{
		ID:     "room",
		Status: RoomInGame,
		Players: []*Player{
			{ID: "p1", Nickname: "A", Connected: true, SessionToken: "tok-a"},
			{ID: "p2", Nickname: "B", Connected: false, SessionToken: "tok-b"},
		},
		Sessions: map[string]string{"tok-a": "p1", "tok-b": "p2"},
		Game: &GameState{
			Phase: PhasePaused,
			Stall: &StallState{
				DisconnectedPlayerID: "p2",
				DisconnectedNickname: "B",
				ReconnectDeadline:    time.Now().UTC().Add(2 * time.Minute),
			},
		},
	}
	manager.rooms[room.ID] = room

	_, player, err := manager.JoinRoom(room.ID, "B")
	if err != nil {
		t.Fatalf("JoinRoom error = %v", err)
	}
	if player.ID != "p2" {
		t.Fatalf("player = %s, want p2", player.ID)
	}
	if player.SessionToken == "tok-b" {
		t.Fatal("expected fresh session token")
	}
}
