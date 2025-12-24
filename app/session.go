package app

import (
	"html/template"
	"time"

	"github.com/zorcal/its-a-me-zorcal/pkg/session"
)

type terminalSessionEntry struct {
	Command   string
	Output    template.HTML
	Error     bool
	Timestamp time.Time
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
