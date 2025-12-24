package app

import (
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/zorcal/its-a-me-zorcal/pkg/session"
)

type terminalSessionEntry struct {
	Command   string
	Output    template.HTML
	Error     bool
	Timestamp time.Time
	Prompt    string
}

func newSessionManager() *session.Manager[terminalSessionEntry] {
	return session.NewManager[terminalSessionEntry](1000)
}

func newTerminalSessionEntry(command string, output template.HTML, isError bool) terminalSessionEntry {
	return terminalSessionEntry{
		Command:   command,
		Output:    output,
		Error:     isError,
		Timestamp: time.Now(),
		Prompt:    "", // Will be set when needed
	}
}

func startSessionCleanupTicker(sessionMgr *session.Manager[terminalSessionEntry]) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			sessionMgr.CleanupOldSessions(24 * time.Hour)
		}
	}()
}

type sessionAdapter struct {
	mgr    *session.Manager[terminalSessionEntry]
	dirs   map[string]string
	dirsMu sync.RWMutex
}

func newSessionAdapter(sessionMgr *session.Manager[terminalSessionEntry]) *sessionAdapter {
	return &sessionAdapter{
		mgr:  sessionMgr,
		dirs: make(map[string]string),
	}
}

// GetCurrentDir implements termui.SessionManager.
func (sa *sessionAdapter) GetCurrentDir(sessionID string) string {
	sa.dirsMu.RLock()
	defer sa.dirsMu.RUnlock()

	if dir, exists := sa.dirs[sessionID]; exists {
		return dir
	}
	return "home/guest" // default
}

// SetCurrentDir implements termui.SessionManager.
func (sa *sessionAdapter) SetCurrentDir(sessionID, dir string) {
	sa.dirsMu.Lock()
	defer sa.dirsMu.Unlock()
	sa.dirs[sessionID] = dir
}

func getSessionID(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setSessionCookie(w http.ResponseWriter, sessionID string) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60, // 24 hours
	}
	http.SetCookie(w, cookie)
}
