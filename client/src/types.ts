export type Suit = "spades" | "hearts" | "diamonds" | "clubs";

export type Card = {
  suit: Suit;
  rank: number;
};

export type Player = {
  playerId: string;
  nickname: string;
  joinOrder: number;
  seatIndex: number;
  isHost: boolean;
  ready: boolean;
  connected: boolean;
  cardsRemaining: number;
  finished: boolean;
};

export type TableCard = {
  playerId: string;
  card: Card;
};

export type GameState = {
  gameId: string;
  phase: "lobby" | "awaiting_turn" | "round_end" | "game_over" | "paused";
  currentTurnPlayerId: string;
  leadPlayerId: string;
  loserPlayerId: string;
  winnerPlayerId: string;
  finishedOrder: string[];
  round: {
    leadPlayerId: string;
    activeSuit: Suit | "";
    tableCards: TableCard[];
    discardCount: number;
  };
  stall?: {
    disconnectedPlayerId: string;
    disconnectedNickname: string;
    message: string;
    reconnectDeadline: string;
  };
  lastEvent: string;
};

export type PublicRoom = {
  roomId: string;
  status: "lobby" | "in_game" | "finished";
  hostPlayerId: string;
  minPlayers: number;
  maxPlayers: number;
  createdAt: string;
  players: Player[];
  game?: GameState;
  readyCount: number;
  playerCount: number;
};

export type PublicState = {
  room: PublicRoom;
  privateHand?: {
    playerId: string;
    hand: Card[];
  };
};

export type Session = {
  roomId: string;
  playerId: string;
  sessionToken: string;
};
