package hub

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
)

type Message struct {
	Kind    string `json:"kind"`
	Payload any    `json:"payload,omitempty"`
}

type Client struct {
	PlayerID string
	Send     chan []byte
}

type FilterFunc func(playerID string, msg Message) (Message, bool)

type Hub struct {
	mu         sync.RWMutex
	clients    map[string]*Client
	broadcast  chan filteredMsg
	register   chan *Client
	unregister chan *Client
}

type filteredMsg struct {
	msg    Message
	filter FilterFunc
}

func New() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan filteredMsg, 64),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c.PlayerID] = c
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c.PlayerID]; ok {
				delete(h.clients, c.PlayerID)
				close(c.Send)
			}
			h.mu.Unlock()
		case fm := <-h.broadcast:
			h.mu.RLock()
			for id, c := range h.clients {
				msg := fm.msg
				if fm.filter != nil {
					var ok bool
					msg, ok = fm.filter(id, fm.msg)
					if !ok {
						continue
					}
				}
				data, err := json.Marshal(msg)
				if err != nil {
					slog.Error("hub marshal error", "err", err)
					continue
				}
				select {
				case c.Send <- data:
				default:
					// slow client — drop
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(c *Client) {
	h.register <- c
}

func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (h *Hub) Broadcast(msg Message) {
	h.broadcast <- filteredMsg{msg: msg}
}

func (h *Hub) BroadcastFiltered(msg Message, filter FilterFunc) {
	h.broadcast <- filteredMsg{msg: msg, filter: filter}
}

func (h *Hub) Send(playerID string, msg Message) {
	h.mu.RLock()
	c, ok := h.clients[playerID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.Send <- data:
	default:
	}
}
