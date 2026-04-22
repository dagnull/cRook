package room

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type Manager struct {
	mu      sync.RWMutex
	rooms   map[string]*Room
	counter atomic.Uint64
}

func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room)}
}

func (m *Manager) Create(name string) *Room {
	id := fmt.Sprintf("room-%d", m.counter.Add(1))
	r := New(id, name)
	m.mu.Lock()
	m.rooms[id] = r
	m.mu.Unlock()
	go r.RunHub(context.Background())
	return r
}

func (m *Manager) Get(id string) (*Room, bool) {
	m.mu.RLock()
	r, ok := m.rooms[id]
	m.mu.RUnlock()
	return r, ok
}

func (m *Manager) List() []*Room {
	m.mu.RLock()
	out := make([]*Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		out = append(out, r)
	}
	m.mu.RUnlock()
	return out
}

func (m *Manager) Delete(id string) {
	m.mu.Lock()
	delete(m.rooms, id)
	m.mu.Unlock()
}
