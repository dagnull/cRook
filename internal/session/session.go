package session

import (
	"context"
	"net/http"

	"github.com/gorilla/securecookie"
)

type contextKey struct{}

const cookieName = "crook_session"

type Session struct {
	PlayerID string
	Name     string
}

type Manager struct {
	sc *securecookie.SecureCookie
}

func NewManager(secretKey string) *Manager {
	key := []byte(secretKey)
	// pad or truncate to 32 bytes for AES-256
	padded := make([]byte, 32)
	copy(padded, key)
	return &Manager{sc: securecookie.New(key, padded)}
}

func (m *Manager) Set(w http.ResponseWriter, s Session) error {
	encoded, err := m.sc.Encode(cookieName, s)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (m *Manager) Get(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return Session{}, false
	}
	var s Session
	if err := m.sc.Decode(cookieName, cookie.Value, &s); err != nil {
		return Session{}, false
	}
	return s, true
}

func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, _ := m.Get(r)
		ctx := context.WithValue(r.Context(), contextKey{}, s)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func FromContext(ctx context.Context) Session {
	s, _ := ctx.Value(contextKey{}).(Session)
	return s
}
