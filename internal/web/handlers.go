package web

import (
	"encoding/json"
	"net/http"

	"github.com/dagnull/cRook/internal/room"
	"github.com/dagnull/cRook/internal/session"
)

type handlers struct {
	mgr      *room.Manager
	sessions *session.Manager
}

func (h *handlers) home(w http.ResponseWriter, r *http.Request) {
	sess := session.FromContext(r.Context())
	rooms := h.mgr.List()
	renderTemplate(w, "home.html", map[string]any{
		"Session": sess,
		"Rooms":   rooms,
	})
}

func (h *handlers) setSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	existing := session.FromContext(r.Context())
	playerID := existing.PlayerID
	if playerID == "" {
		playerID = newID()
	}

	if err := h.sessions.Set(w, session.Session{PlayerID: playerID, Name: body.Name}); err != nil {
		http.Error(w, "could not set session", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) createRoom(w http.ResponseWriter, r *http.Request) {
	sess := session.FromContext(r.Context())
	if sess.PlayerID == "" {
		http.Error(w, "set a name first", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		name = sess.Name + "'s room"
	}
	rm := h.mgr.Create(name)
	http.Redirect(w, r, "/rooms/"+rm.ID, http.StatusSeeOther)
}

func (h *handlers) roomPage(w http.ResponseWriter, r *http.Request) {
	id := roomID(r)
	rm, ok := h.mgr.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	sess := session.FromContext(r.Context())
	renderTemplate(w, "room.html", map[string]any{
		"Session": sess,
		"Room":    rm,
	})
}
