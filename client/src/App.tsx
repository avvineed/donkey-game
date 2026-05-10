import { useEffect, useRef, useState } from "react";
import { createRoom, joinRoom, openRoomSocket, playCard, reconnect, setReady, startGame } from "./api";
import { canPlayCard, cardId, formatCard, playerLabel, sortHand } from "./gameRules";
import cardBack from "./assets/card-back.svg";
import type { Card, PublicState, Session } from "./types";

const STORAGE_KEY = "kazhuta-session";

type Screen = "auth" | "lobby" | "table";

export function App() {
  const [screen, setScreen] = useState<Screen>("auth");
  const [nickname, setNickname] = useState("");
  const [roomCode, setRoomCode] = useState("");
  const [session, setSession] = useState<Session | null>(loadSession());
  const [state, setState] = useState<PublicState | null>(null);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!session) {
      return;
    }
    reconnect(session.roomId, session.sessionToken)
      .then((payload) => {
        setSession({
          roomId: payload.state.room.roomId,
          playerId: payload.playerId,
          sessionToken: payload.sessionToken
        });
        setState(payload.state);
        setScreen(payload.state.room.status === "lobby" ? "lobby" : "table");
      })
      .catch(() => {
        clearSession();
        setSession(null);
      });
  }, []);

  useEffect(() => {
    if (!session) {
      return;
    }
    socketRef.current?.close();
    const socket = openRoomSocket(session.roomId, session.playerId, (nextState) => {
      setState(nextState);
      setScreen(nextState.room.status === "lobby" ? "lobby" : "table");
    });
    socketRef.current = socket;
    return () => socket.close();
  }, [session?.roomId, session?.playerId]);

  async function handleCreateRoom() {
    if (nickname.trim().length < 2) {
      setError("Nickname must be at least 2 characters.");
      return;
    }
    setError("");
    setSubmitting(true);
    try {
      const payload = await createRoom(nickname.trim());
      const nextSession = {
        roomId: payload.state.room.roomId,
        playerId: payload.playerId,
        sessionToken: payload.sessionToken
      };
      saveSession(nextSession);
      setSession(nextSession);
      setState(payload.state);
      setScreen("lobby");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to create room");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleJoinRoom() {
    if (nickname.trim().length < 2 || !roomCode.trim()) {
      setError("Enter a nickname and room ID.");
      return;
    }
    setError("");
    setSubmitting(true);
    try {
      const payload = await joinRoom(roomCode.trim(), nickname.trim());
      const nextSession = {
        roomId: payload.state.room.roomId,
        playerId: payload.playerId,
        sessionToken: payload.sessionToken
      };
      saveSession(nextSession);
      setSession(nextSession);
      setState(payload.state);
      setScreen("lobby");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to join room");
    } finally {
      setSubmitting(false);
    }
  }

  async function toggleReady() {
    if (!session || !state) {
      return;
    }
    const localPlayer = state.room.players.find((player) => player.playerId === session.playerId);
    if (!localPlayer) {
      return;
    }
    setSubmitting(true);
    try {
      await setReady(session.roomId, session.playerId, !localPlayer.ready);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to update ready state");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleStartGame() {
    if (!session) {
      return;
    }
    setSubmitting(true);
    try {
      await startGame(session.roomId, session.playerId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to start game");
    } finally {
      setSubmitting(false);
    }
  }

  async function handlePlay(card: Card) {
    if (!session || !state || !canPlayCard(state, session.playerId, card) || submitting) {
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      await playCard(session.roomId, session.playerId, card);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to play card");
    } finally {
      setSubmitting(false);
    }
  }

  function handleLeave() {
    socketRef.current?.close();
    clearSession();
    setSession(null);
    setState(null);
    setScreen("auth");
    setError("");
  }

  if (screen === "auth" || !state || !session) {
    return (
      <main className="shell auth-shell">
        <section className="hero">
          <div className="brand-row">
            <span className="brand-mark">K</span>
            <p className="eyebrow">KazhutaKali</p>
          </div>
          <h1>Kazhuta</h1>
          <p className="subtle">
            A private card table for 3 to 13 players, tuned for the sweet chaos of 4 to 6.
          </p>
          <div className="hero-tableau" aria-hidden="true">
            <img src={cardBack} alt="" className="hero-card hero-card-back" />
            <div className="hero-card hero-card-face">
              <span>A♠</span>
              <strong>Kazhuta</strong>
            </div>
            <div className="hero-chip">52</div>
          </div>
        </section>
        <section className="panel auth-panel">
          <label>
            Nickname
            <input value={nickname} onChange={(event) => setNickname(event.target.value)} maxLength={24} />
          </label>
          <div className="auth-actions">
            <button onClick={handleCreateRoom} disabled={submitting}>
              Create room
            </button>
          </div>
          <div className="divider">Join a table</div>
          <label>
            Room ID
            <input value={roomCode} onChange={(event) => setRoomCode(event.target.value)} />
          </label>
          <button className="secondary" onClick={handleJoinRoom} disabled={submitting}>
            Join room
          </button>
          {error ? <p className="error">{error}</p> : null}
        </section>
      </main>
    );
  }

  const localPlayer = state.room.players.find((player) => player.playerId === session.playerId);
  const canStart =
    localPlayer?.isHost &&
    state.room.playerCount >= state.room.minPlayers &&
    state.room.players.every((player) => player.ready);

  return (
    <main className="shell app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Private Table</p>
          <h2>{state.room.roomId}</h2>
        </div>
        <button className="secondary" onClick={handleLeave}>
          Leave
        </button>
      </header>

      {screen === "lobby" ? (
        <section className="lobby-grid">
          <article className="panel lobby-panel">
            <div className="section-heading">
              <p className="eyebrow">Lobby</p>
              <h3>Players at the veranda</h3>
            </div>
            <p className="subtle">
              {state.room.playerCount}/{state.room.maxPlayers} seated · {state.room.readyCount} ready · best with 4 to 6
            </p>
            <ul className="player-list">
              {state.room.players.map((player) => (
                <li key={player.playerId}>
                  <span className="player-meta">
                    <img src={cardBack} alt="" className="card-back-mini" />
                    {playerLabel(player, session.playerId)}
                  </span>
                  <span className={`status-chip ${player.ready ? "ready" : ""}`}>{player.ready ? "Ready" : "Waiting"}</span>
                </li>
              ))}
            </ul>
            <div className="lobby-actions">
              <button onClick={toggleReady} disabled={submitting}>
                {localPlayer?.ready ? "Unready" : "Ready up"}
              </button>
              {localPlayer?.isHost ? (
                <button className="secondary" onClick={handleStartGame} disabled={!canStart || submitting}>
                  Start game
                </button>
              ) : null}
            </div>
          </article>
          <article className="panel rules-panel">
            <div className="section-heading">
              <p className="eyebrow">Kazhuta rules</p>
              <h3>Follow suit, shed cards</h3>
            </div>
            <ul className="rules-list">
              <li>The Ace of Spades must open the game.</li>
              <li>You must follow suit when you can.</li>
              <li>A legal off-suit play is a strike only when you have no active-suit card.</li>
              <li>The last player still holding cards becomes the Kazhuta.</li>
            </ul>
            {error ? <p className="error">{error}</p> : null}
          </article>
        </section>
      ) : (
        <GameTable
          state={state}
          session={session}
          error={error}
          submitting={submitting}
          onPlay={handlePlay}
        />
      )}
    </main>
  );
}

type GameTableProps = {
  state: PublicState;
  session: Session;
  error: string;
  submitting: boolean;
  onPlay: (card: Card) => void;
};

function GameTable({ state, session, error, submitting, onPlay }: GameTableProps) {
  const hand = sortHand(state.privateHand?.hand ?? []);
  const game = state.room.game;
  if (!game) {
    return null;
  }

  const tableCards = game.round.tableCards ?? [];
  const finishedOrder = game.finishedOrder ?? [];
  const recentActions = game.recentActions ?? [];
  const currentTurnId = game.currentTurnPlayerId;
  const stallDeadline = game.stall ? new Date(game.stall.reconnectDeadline).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : "";

  return (
    <section className="table-grid">
      <article className="panel table-panel">
        {game.phase === "paused" && game.stall ? (
          <div className="stall-banner" role="alert">
            <strong>{game.stall.message}</strong>
            <span>Rejoin window stays open until {stallDeadline}.</span>
          </div>
        ) : null}
        <div className="status-row">
          <div>
            <p className="eyebrow">Turn</p>
            <h3>{nameFor(state, game.currentTurnPlayerId, session.playerId)}</h3>
          </div>
          <div>
            <p className="eyebrow">Suit</p>
            <h3>{game.round.activeSuit || "Unset"}</h3>
          </div>
          <div>
            <p className="eyebrow">Discarded</p>
            <h3>{game.round.discardCount}</h3>
          </div>
        </div>
        <div className="table-cards">
          {tableCards.length === 0 ? <p className="subtle">No cards on the table yet.</p> : null}
          {tableCards.map((entry) => (
            <div key={`${entry.playerId}-${cardId(entry.card)}`} className="table-card">
              <span className={`table-rank ${entry.card.suit === "hearts" || entry.card.suit === "diamonds" ? "red" : ""}`}>
                {formatCard(entry.card)}
              </span>
              <small>{nameFor(state, entry.playerId, session.playerId)}</small>
            </div>
          ))}
        </div>
        <div className="recent-actions">
          <div className="section-heading">
            <p className="eyebrow">Recent play</p>
            <h3>Latest cards and cuts</h3>
          </div>
          {recentActions.length === 0 ? <p className="subtle">Recent actions will appear here as the round progresses.</p> : null}
          <ul className="activity-list">
            {recentActions.map((action, index) => (
              <li key={`${action.type}-${action.playerId}-${index}`} className={action.type === "cut" ? "activity-cut" : ""}>
                <span>{action.message}</span>
              </li>
            ))}
          </ul>
        </div>
        {game.phase === "game_over" ? (
          <div className="result-banner">
            <strong>{nameFor(state, game.loserPlayerId, session.playerId)}</strong> is the Kazhuta.
          </div>
        ) : null}
      </article>
      <aside className="panel sidebar-panel">
        <div className="section-heading">
          <p className="eyebrow">Seats</p>
          <h3>Players</h3>
        </div>
        <ul className="player-list">
          {state.room.players.map((player) => (
            <li
              key={player.playerId}
              className={player.playerId === currentTurnId && game.phase !== "game_over" ? "current-turn-player" : ""}
            >
              <span className="player-meta">
                <img src={cardBack} alt="" className="card-back-mini" />
                {playerLabel(player, session.playerId)}
              </span>
              <span>
                {player.playerId === currentTurnId && game.phase !== "game_over" ? "Turn · " : ""}
                {player.finished ? "Finished" : `${player.cardsRemaining} cards`}
                {!player.connected ? " · offline" : ""}
              </span>
            </li>
          ))}
        </ul>
        <h3 className="side-subhead">Finish order</h3>
        <ol className="finish-list">
          {finishedOrder.map((playerId) => (
            <li key={playerId}>{nameFor(state, playerId, session.playerId)}</li>
          ))}
        </ol>
      </aside>
      <article className="panel hand-panel">
        <div className="hand-header">
          <div>
            <p className="eyebrow">Private hand</p>
            <h3>Your cards</h3>
          </div>
          {error ? <p className="error">{error}</p> : null}
        </div>
        <div className="hand-grid">
          {hand.map((card) => {
            const playable = canPlayCard(state, session.playerId, card);
            return (
              <button
                key={cardId(card)}
                className={`card-button ${playable ? "playable" : "locked"}`}
                onClick={() => onPlay(card)}
                disabled={!playable || submitting || game.phase === "game_over"}
              >
                <span className={`card-corner ${card.suit === "hearts" || card.suit === "diamonds" ? "red" : ""}`}>
                  {formatCard(card)}
                </span>
                <span className={`card-center ${card.suit === "hearts" || card.suit === "diamonds" ? "red" : ""}`}>
                  {formatCard(card)}
                </span>
              </button>
            );
          })}
        </div>
      </article>
    </section>
  );
}

function saveSession(session: Session) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
}

function loadSession(): Session | null {
  const value = localStorage.getItem(STORAGE_KEY);
  if (!value) {
    return null;
  }
  try {
    return JSON.parse(value) as Session;
  } catch {
    return null;
  }
}

function clearSession() {
  localStorage.removeItem(STORAGE_KEY);
}

function nameFor(state: PublicState, playerId: string, localPlayerId: string) {
  const player = state.room.players.find((entry) => entry.playerId === playerId);
  return player ? playerLabel(player, localPlayerId) : "Unknown";
}
