# Kazhuta Client

This folder contains the React frontend for Kazhuta. It handles room creation, room join/rejoin, lobby flow, the live game table, and player-facing card interactions.

## Stack

- React `18`
- Vite
- TypeScript
- Vitest

## Responsibilities

- Create or join a room with a nickname
- Reconnect a player session from local state
- Show the lobby and ready/start controls
- Render the live table and the local player hand
- Highlight the active turn
- Show stalled-game alerts when a player leaves
- Prevent obviously illegal moves in the UI before the server re-validates them

## Scripts

Install dependencies:

```bash
cd client
npm install
```

Start the dev server:

```bash
cd client
npm run dev
```

Build for production:

```bash
cd client
npm run build
```

Run frontend tests:

```bash
cd client
npm test
```

From the repo root you can also use:

```bash
make install
make start-client
make test-client
make build-client
```

## Expected backend

The frontend currently expects the backend at:

`http://localhost:8080`

WebSocket updates are opened through:

`ws://localhost:8080/ws`

## UI behavior

- Only the local player sees their own cards
- Opponents are represented by name, status, and card count
- The current turn is highlighted in the player list
- If a player leaves during a live game, the UI shows a stalled-game banner with the reconnect deadline

## Key files

- `src/App.tsx` - main app flow and table UI
- `src/api.ts` - HTTP and WebSocket client
- `src/gameRules.ts` - client-side move gating
- `src/types.ts` - shared frontend state types
- `src/styles.css` - game theme and layout
