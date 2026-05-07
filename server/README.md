# Kazhuta Server

This folder contains the Go backend for Kazhuta. It exposes HTTP endpoints for room and game actions and uses WebSockets for live room updates.

## Responsibilities

- Create and manage rooms
- Track players, readiness, and reconnect sessions
- Start and run live games
- Enforce turn order and card rules
- Broadcast state updates to connected clients
- Stall active games when a player disconnects and allow reconnect within the grace window

## Entry point

Run the server directly:

```bash
cd server
go run ./cmd/server
```

By default it listens on `:8080`.

You can override the bind address:

```bash
ADDR=:9090 go run ./cmd/server
```

## Main packages

- `cmd/server` - application entry point
- `internal/api` - HTTP routes and WebSocket handling
- `internal/game` - room manager, game engine, game state types, tests

## API surface

Current routes:

- `GET /health`
- `POST /api/rooms`
- `POST /api/rooms/join`
- `POST /api/rooms/reconnect`
- `POST /api/rooms/ready`
- `POST /api/rooms/start`
- `POST /api/games/play`
- `GET /ws`

## Reconnect and stalled game behavior

When a player disconnects during a live game:

- the game phase is moved to `paused`
- all clients receive the stalled state over WebSocket
- the missing player gets a `3 minute` reconnect window
- the player can reclaim the same seat by rejoining the same room with the same nickname
- when all players are connected again, the game resumes automatically
- if the window expires, the stalled game is ended

## Tests

Run backend tests:

```bash
cd server
go test ./...
```

From the repo root you can also use:

```bash
make test-server
```
