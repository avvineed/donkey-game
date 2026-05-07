import type { Card, Player, PublicState, Suit } from "./types";

const ACE_OF_SPADES: Card = { suit: "spades", rank: 14 };

export function cardId(card: Card) {
  return `${card.suit}-${card.rank}`;
}

export function canPlayCard(state: PublicState, localPlayerId: string, card: Card) {
  const game = state.room.game;
  const hand = state.privateHand?.hand ?? [];
  if (!game || game.phase !== "awaiting_turn") {
    return false;
  }
  if (game.currentTurnPlayerId !== localPlayerId) {
    return false;
  }
  if (!hand.some((item) => item.suit === card.suit && item.rank === card.rank)) {
    return false;
  }
  if (game.round.tableCards.length === 0 && game.round.discardCount === 0) {
    return card.suit === ACE_OF_SPADES.suit && card.rank === ACE_OF_SPADES.rank;
  }
  if (!game.round.activeSuit) {
    return true;
  }
  const hasActiveSuit = hand.some((item) => item.suit === game.round.activeSuit);
  if (!hasActiveSuit) {
    return true;
  }
  return card.suit === game.round.activeSuit;
}

export function playerLabel(player: Player, localPlayerId: string) {
  return player.playerId === localPlayerId ? `${player.nickname} (You)` : player.nickname;
}

export function formatCard(card: Card) {
  const ranks: Record<number, string> = {
    11: "J",
    12: "Q",
    13: "K",
    14: "A"
  };
  const symbols: Record<Suit, string> = {
    spades: "♠",
    hearts: "♥",
    diamonds: "♦",
    clubs: "♣"
  };
  return `${ranks[card.rank] ?? card.rank}${symbols[card.suit]}`;
}

export function sortHand(cards: Card[]) {
  const suitWeight: Record<Suit, number> = {
    spades: 0,
    hearts: 1,
    diamonds: 2,
    clubs: 3
  };
  return [...cards].sort((left, right) => {
    if (left.suit === right.suit) {
      return left.rank - right.rank;
    }
    return suitWeight[left.suit] - suitWeight[right.suit];
  });
}
