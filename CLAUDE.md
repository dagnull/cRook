# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**cRook** is a multiplayer web app for the card game Rook, written in Go 1.26.
Module path: `github.com/dagnull/cRook`

Players connect via WebSocket. The server maintains authoritative game state; clients are driven entirely by server events and a `state_sync` message sent after every action.

## Commands

```bash
# Build
go build ./...

# Run (serves on :8080 by default)
go run .

# Test all packages
go test ./...

# Test a specific package or test
go test ./internal/game/... -v -run TestFullGame

# Format
go fmt ./...
```

## Architecture

```
main.go                     Entry point; embeds ui/ FS; wires config, sessions, rooms, router
internal/
  config/      config.go    Env-based config (port, secret key)
  session/     session.go   Cookie-backed sessions (gorilla/securecookie)
  hub/         hub.go       Per-room WebSocket broadcast hub (fan-out, filtered sends)
  game/
    card.go                 Card, Suit (MarshalJSON → "yellow"/"red"/"green"/"black"/"none")
    deck.go                 NewDeck(), Shuffle()
    state.go                GameState, PlayerState, Trick, Play, GameOptions
    action.go               Action, ActionKind constants
    event.go                Event, EventKind constants, all payload structs
    phase.go                Phase type and constants
    ruleset.go              RuleSet interface (lives here to avoid import cycle)
    engine.go               Engine — stateless; Deal/Apply entry points
    rules/
      standard.go           Standard 4-player Rook rules (implements RuleSet)
  room/
    manager.go              RoomManager — create/get rooms
    room.go                 Room — holds GameState, Hub; StartGame/Dispatch/StateSyncFor
  web/
    server.go               HTTP server with graceful shutdown
    routes.go               chi router wiring
    handlers.go             HTTP page handlers
    ws.go                   WebSocket upgrade, read/write loops, sendStateSync
    action.go               Parse incoming WS JSON → game.Action
    template.go             Per-page template instances (avoids {{define}} collision)
    static.go               Embedded static file serving
    util.go                 Shared HTTP helpers
ui/
  templates/
    base.html               Shared layout (Alpine.js + style.css loaded here)
    home.html               Lobby page
    room.html               Game room page (all game UI)
    partials/room_card.html Room list card
  static/
    app.js                  Alpine.js gameRoom() component — all client logic
    style.css               Dark theme + toast animations
```

## Key design decisions

- **Stateless engine**: `Engine.Apply(state, action) → (state, events, error)`. The `Room` owns state; the engine is pure and never mutates shared data.
- **RuleSet interface** lives in `internal/game` (not `internal/game/rules`) to avoid the import cycle `engine → rules → game`.
- **Per-page templates**: `template.go` creates a separate `*template.Template` per page so `{{define "content"}}` blocks don't overwrite each other.
- **Embed at root**: `//go:embed ui` is in `main.go`; sub-packages receive `fs.FS` via `fs.Sub()` because Go embed paths can't traverse `..`.
- **Hub reconnect safety**: `Hub.Unregister` checks pointer equality before deleting the map entry so a reconnecting player's new client isn't evicted by the old connection's deferred unregister.
- **State sync on every action**: `Room.Dispatch` calls `broadcastStateSyncs()` after every action, sending each player a personalised `state_sync` (hand visible only to owner, running trick points from CompletedTricks).
- **`pendingAction` guard**: The client sets `pendingAction = true` on every `sendAction` call and clears it only on `state_sync`, preventing double-sends from rapid clicks.

## Card encoding

- `Suit` serialises as a string: `"yellow"`, `"red"`, `"green"`, `"black"`, `"none"`.
- The Rook card is `{suit: "none", value: 0}`. The client checks `value === 0` before parsing the suit.
- Point values: 5 → 5 pts, 10 → 10 pts, 14 → 10 pts, 1 → 15 pts, Rook → 20 pts.

## WebSocket message flow

**Server → client**
| Kind | When |
|---|---|
| `state_sync` | After every action and on (re)connect |
| `player_joined` / `player_left` | On WS connect/disconnect |
| `card_dealt` | On deal (filtered: other players get empty card list) |
| `game_started` | On deal |
| `bid_placed` / `player_passed` / `bidding_won` | During bidding |
| `trump_named` / `nest_set` | During nesting |
| `card_played` / `trick_won` | During playing |
| `round_scored` | End of round |
| `error` | Action rejected |

**Client → server**
`start_game` · `bid` · `pass` · `name_trump` · `set_nest` · `play_card`