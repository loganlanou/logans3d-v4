package session

import (
	"encoding/gob"
	"fmt"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
)

const (
	sessionName = "logans3d_session"
	userKey     = "user"
)

// Manager manages user sessions
type Manager struct {
	store sessions.Store
}

// NewManager creates a new session manager
func NewManager(secret string) *Manager {
	// Register UserData type for gob encoding
	gob.Register(&UserData{})

	// Create cookie store with secret key
	store := sessions.NewCookieStore([]byte(secret))

	// Configure session options
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: 2,     // Lax mode
	}

	return &Manager{
		store: store,
	}
}

// CreateSession creates a new session with user data
func (m *Manager) CreateSession(c echo.Context, user *UserData) error {
	session, err := m.store.Get(c.Request(), sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.Values[userKey] = user

	if err := session.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// GetSession retrieves user data from the session
func (m *Manager) GetSession(c echo.Context) (*UserData, error) {
	session, err := m.store.Get(c.Request(), sessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	userData, ok := session.Values[userKey].(*UserData)
	if !ok || userData == nil {
		return nil, fmt.Errorf("no user data in session")
	}

	return userData, nil
}

// DestroySession clears the session
func (m *Manager) DestroySession(c echo.Context) error {
	session, err := m.store.Get(c.Request(), sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Mark session for deletion
	session.Options.MaxAge = -1
	delete(session.Values, userKey)

	if err := session.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("failed to destroy session: %w", err)
	}

	return nil
}
