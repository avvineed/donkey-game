import type { Card, PublicState } from "./types";

const API_BASE = "http://localhost:8080";

async function request<T>(path: string, init: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      "Content-Type": "application/json"
    },
    ...init
  });

  const payload = (await response.json()) as T & { error?: string };
  if (!response.ok) {
    throw new Error(payload.error || "Request failed");
  }
  return payload;
}

function normalizeState(state: PublicState): PublicState {
  return {
    ...state,
    room: {
      ...state.room,
      players: state.room.players ?? [],
      game: state.room.game
        ? {
            ...state.room.game,
            finishedOrder: state.room.game.finishedOrder ?? [],
            round: {
              ...state.room.game.round,
              tableCards: state.room.game.round.tableCards ?? []
            },
            stall: state.room.game.stall
              ? {
                  ...state.room.game.stall
                }
              : undefined
          }
        : undefined
    },
    privateHand: state.privateHand
      ? {
          ...state.privateHand,
          hand: state.privateHand.hand ?? []
        }
      : undefined
  };
}

export async function createRoom(nickname: string) {
  const response = await request<{
    playerId: string;
    sessionToken: string;
    state: PublicState;
  }>("/api/rooms", {
    method: "POST",
    body: JSON.stringify({ nickname })
  });
  return { ...response, state: normalizeState(response.state) };
}

export async function joinRoom(roomId: string, nickname: string) {
  const response = await request<{
    playerId: string;
    sessionToken: string;
    state: PublicState;
  }>("/api/rooms/join", {
    method: "POST",
    body: JSON.stringify({ roomId, nickname })
  });
  return { ...response, state: normalizeState(response.state) };
}

export async function reconnect(roomId: string, sessionToken: string) {
  const response = await request<{
    playerId: string;
    sessionToken: string;
    state: PublicState;
  }>("/api/rooms/reconnect", {
    method: "POST",
    body: JSON.stringify({ roomId, sessionToken })
  });
  return { ...response, state: normalizeState(response.state) };
}

export async function setReady(roomId: string, playerId: string, ready: boolean) {
  return request<{ status: string }>("/api/rooms/ready", {
    method: "POST",
    body: JSON.stringify({ roomId, playerId, ready })
  });
}

export async function startGame(roomId: string, playerId: string) {
  return request<{ status: string }>("/api/rooms/start", {
    method: "POST",
    body: JSON.stringify({ roomId, playerId })
  });
}

export async function playCard(roomId: string, playerId: string, card: Card) {
  return request<{ status: string }>("/api/games/play", {
    method: "POST",
    body: JSON.stringify({ roomId, playerId, card })
  });
}

export function openRoomSocket(roomId: string, playerId: string, onState: (state: PublicState) => void) {
  const socket = new WebSocket(`ws://localhost:8080/ws?roomId=${roomId}&playerId=${playerId}`);
  socket.addEventListener("message", (event) => {
    const envelope = JSON.parse(event.data) as { type: string; data: PublicState };
    if (envelope.type === "state_sync") {
      onState(normalizeState(envelope.data));
    }
  });
  return socket;
}
