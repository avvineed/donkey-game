# Kazhuta

Kazhuta is an online multiplayer implementation of the Kazhuta/Kazhutakali card game. The app uses a Go backend for lobby and game state management, and a React frontend for the room, table, and live gameplay UI.

## Repo layout

- `server/` - Go HTTP and WebSocket server, in-memory room manager, game engine
- `client/` - React + Vite frontend
- `Makefile` - repo-level commands for install, start, stop, test, and build

## Requirements

- Go `1.20+`
- Node.js `20+`
- npm `10+`

## Recommended workflow

Install frontend dependencies once:

```bash
make install
```

Start both apps in the background:

```bash
make start
```

That starts:

- backend at `http://127.0.0.1:8080`
- frontend at `http://127.0.0.1:5173`

Stop both:

```bash
make stop
```

Restart both:

```bash
make restart
```

Check whether they are running:

```bash
make status
```

Run tests:

```bash
make test
```

Build the frontend:

```bash
make build
```

Clean local runtime files created by the Makefile:

```bash
make clean-run
```

## Makefile notes

The Makefile stores local PID files and logs in `.run/`.

- server PID: `.run/server.pid`
- client PID: `.run/client.pid`
- server log: `.run/server.log`
- client log: `.run/client.log`

This lets you start and stop the app from one place without keeping two terminals open.

## Gameplay notes

- Supports `3` to `13` players
- Best experience is currently tuned around `4` to `6` players
- The server is authoritative for dealing, turns, suit-follow validation, strikes, and finish order
- If a player disconnects during a live game, the room is stalled for a `3 minute` reconnect window
- A disconnected player can reclaim their seat during the stall window by joining the same room again with the same nickname

## More docs

- backend details: [server/README.md](/Users/vineed/Developer/rnd/games/server/README.md)
- frontend details: [client/README.md](/Users/vineed/Developer/rnd/games/client/README.md)
