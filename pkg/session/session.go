package session

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultHistoryLimit = 100

// Manager handles multiple sessions with configurable history limits.
type Manager[T any] struct {
	sessions     map[string]*Session[T]
	historyLimit int
	mu           sync.RWMutex
}

// NewManager creates a session manager with the given history limit.
func NewManager[T any](historyLimit int) *Manager[T] {
	if historyLimit <= 0 {
		historyLimit = defaultHistoryLimit
	}
	return &Manager[T]{
		sessions:     make(map[string]*Session[T]),
		historyLimit: historyLimit,
	}
}

// GetOrCreateSession returns an existing session or creates a new one.
func (m *Manager[T]) GetOrCreateSession(sessionID string) *Session[T] {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	session, exists := m.sessions[sessionID]
	if !exists {
		session = &Session[T]{
			id:           sessionID,
			history:      make([]T, 0),
			lastUsed:     time.Now(),
			historyLimit: m.historyLimit,
		}
		m.sessions[sessionID] = session
	} else {
		session.lastUsed = time.Now()
	}

	return session
}

// CleanupOldSessions removes sessions older than maxAge.
func (m *Manager[T]) CleanupOldSessions(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, session := range m.sessions {
		if session.lastUsed.Before(cutoff) {
			delete(m.sessions, id)
		}
	}
}

// Session stores typed history entries for a user session.
type Session[T any] struct {
	id           string
	history      []T
	historyLimit int
	lastUsed     time.Time
	mu           sync.RWMutex
}

func (s *Session[T]) ID() string {
	return s.id
}

// AddEntry adds an entry to the session history with automatic trimming.
func (s *Session[T]) AddEntry(entry T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.history = append(s.history, entry)

	if len(s.history) > s.historyLimit {
		s.history = s.history[1:]
	}
}

// History returns a copy of the session's history entries.
func (s *Session[T]) History() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := make([]T, len(s.history))
	copy(history, s.history)
	return history
}

// ClearHistory removes all entries from the session history.
func (s *Session[T]) ClearHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.history = s.history[:0]
}
