package core

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session represents a Claude CLI session
type Session struct {
	ID        string
	Context   map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
	Active    bool
}

// SessionManager manages Claude CLI sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	current  string
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new session
func (sm *SessionManager) CreateSession() (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        sessionID,
		Context:   make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessions[sessionID] = session
	sm.current = sessionID

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

// GetCurrentSession retrieves the current active session
func (sm *SessionManager) GetCurrentSession() (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.current == "" {
		return nil, false
	}

	session, exists := sm.sessions[sm.current]
	return session, exists
}

// SetCurrentSession sets the current active session
func (sm *SessionManager) SetCurrentSession(sessionID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[sessionID]; exists {
		sm.current = sessionID
		return true
	}
	return false
}

// UpdateSessionContext updates the context of a session
func (sm *SessionManager) UpdateSessionContext(sessionID string, key string, value interface{}) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.Context[key] = value
		session.UpdatedAt = time.Now()
		return true
	}
	return false
}

// CloseSession marks a session as inactive
func (sm *SessionManager) CloseSession(sessionID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.Active = false
		session.UpdatedAt = time.Now()
		if sm.current == sessionID {
			sm.current = ""
		}
		return true
	}
	return false
}

// CleanupInactiveSessions removes inactive sessions older than the specified duration
func (sm *SessionManager) CleanupInactiveSessions(maxAge time.Duration) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for id, session := range sm.sessions {
		if !session.Active && session.UpdatedAt.Before(cutoff) {
			delete(sm.sessions, id)
			cleaned++
		}
	}

	return cleaned
}

// ListActiveSessions returns all active sessions
func (sm *SessionManager) ListActiveSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var activeSessions []*Session
	for _, session := range sm.sessions {
		if session.Active {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions
}

// generateSessionID generates a unique session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}