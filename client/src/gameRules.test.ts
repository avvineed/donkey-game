import { describe, expect, it } from "vitest";
import { canPlayCard } from "./gameRules";
import type { PublicState } from "./types";

function stateFixture(): PublicState {
  return {
    room: {
      roomId: "room",
      status: "in_game",
      hostPlayerId: "p1",
      minPlayers: 3,
      maxPlayers: 13,
      createdAt: "",
      readyCount: 3,
      playerCount: 3,
      players: [
        { playerId: "p1", nickname: "A", joinOrder: 0, seatIndex: 0, isHost: true, ready: true, connected: true, cardsRemaining: 2, finished: false },
        { playerId: "p2", nickname: "B", joinOrder: 1, seatIndex: 1, isHost: false, ready: true, connected: true, cardsRemaining: 2, finished: false },
        { playerId: "p3", nickname: "C", joinOrder: 2, seatIndex: 2, isHost: false, ready: true, connected: true, cardsRemaining: 1, finished: false }
      ],
      game: {
        gameId: "game",
        phase: "awaiting_turn",
        currentTurnPlayerId: "p2",
        leadPlayerId: "p1",
        loserPlayerId: "",
        winnerPlayerId: "",
        finishedOrder: [],
        round: {
          leadPlayerId: "p1",
          activeSuit: "spades",
          tableCards: [{ playerId: "p1", card: { suit: "spades", rank: 14 } }],
          discardCount: 0
        },
        lastEvent: "card_played"
      }
    },
    privateHand: {
      playerId: "p2",
      hand: [
        { suit: "spades", rank: 10 },
        { suit: "hearts", rank: 4 }
      ]
    }
  };
}

describe("canPlayCard", () => {
  it("blocks off-suit cards when the player can follow suit", () => {
    const state = stateFixture();
    expect(canPlayCard(state, "p2", { suit: "hearts", rank: 4 })).toBe(false);
    expect(canPlayCard(state, "p2", { suit: "spades", rank: 10 })).toBe(true);
  });

  it("requires ace of spades on opening move", () => {
    const state = stateFixture();
    state.room.game!.round.tableCards = [];
    state.room.game!.round.discardCount = 0;
    state.room.game!.currentTurnPlayerId = "p2";
    state.privateHand!.hand = [
      { suit: "spades", rank: 14 },
      { suit: "hearts", rank: 4 }
    ];
    expect(canPlayCard(state, "p2", { suit: "hearts", rank: 4 })).toBe(false);
    expect(canPlayCard(state, "p2", { suit: "spades", rank: 14 })).toBe(true);
  });
});
