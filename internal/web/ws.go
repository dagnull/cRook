package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"nhooyr.io/websocket"

	"github.com/dagnull/cRook/internal/hub"
	"github.com/dagnull/cRook/internal/room"
	"github.com/dagnull/cRook/internal/session"
)

func (h *handlers) websocket(w http.ResponseWriter, r *http.Request) {
	sess := session.FromContext(r.Context())
	if sess.PlayerID == "" {
		http.Error(w, "set a name first", http.StatusUnauthorized)
		return
	}

	id := roomID(r)
	rm, ok := h.mgr.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("ws accept", "err", err)
		return
	}

	client := &hub.Client{
		PlayerID: sess.PlayerID,
		Send:     make(chan []byte, 64),
	}
	rm.Hub.Register(client)
	defer rm.Hub.Unregister(client)

	rm.Join(room.Player{ID: sess.PlayerID, Name: sess.Name})
	defer rm.Leave(sess.PlayerID)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Write loop: hub → WebSocket.
	go func() {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-client.Send:
				if !ok {
					return
				}
				if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
					return
				}
			}
		}
	}()

	// Announce join before sending state_sync so other clients know someone arrived.
	rm.Hub.Broadcast(hub.Message{
		Kind:    "player_joined",
		Payload: map[string]string{"player_id": sess.PlayerID, "name": sess.Name},
	})

	// Send state_sync directly to this client.
	sendStateSync(rm, sess.PlayerID, client)

	// Read loop: WebSocket → Dispatch.
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}

		action, isStartGame, parseErr := parseAction(data, sess.PlayerID)
		if parseErr != nil {
			sendError(client, parseErr.Error())
			continue
		}

		if isStartGame {
			if err := rm.StartGame(); err != nil {
				sendError(client, err.Error())
			}
			// After starting, send state_sync to every connected player individually.
			// The engine broadcasts card_dealt events which the hub filters, so each
			// player's hand is already sent. The state_sync here syncs phase/bid/etc.
			continue
		}

		if err := rm.Dispatch(action); err != nil {
			sendError(client, err.Error())
		}
	}

	rm.Hub.Broadcast(hub.Message{
		Kind:    "player_left",
		Payload: map[string]string{"player_id": sess.PlayerID, "name": sess.Name},
	})
	conn.Close(websocket.StatusNormalClosure, "")
}

func sendStateSync(rm *room.Room, playerID string, client *hub.Client) {
	payload := rm.StateSyncFor(playerID)
	data, err := json.Marshal(hub.Message{Kind: "state_sync", Payload: payload})
	if err != nil {
		slog.Error("state_sync marshal", "err", err)
		return
	}
	select {
	case client.Send <- data:
	default:
	}
}

func sendError(client *hub.Client, msg string) {
	data, _ := json.Marshal(hub.Message{
		Kind:    "error",
		Payload: map[string]string{"message": msg},
	})
	select {
	case client.Send <- data:
	default:
	}
}
