package termui

import "sync"

type mockSessionManager struct {
	dirs   map[string]string
	dirsMu sync.RWMutex
}

func newMockSessionManager() *mockSessionManager {
	return &mockSessionManager{
		dirs: make(map[string]string),
	}
}

func (m *mockSessionManager) GetCurrentDir(sessionID string) string {
	m.dirsMu.RLock()
	defer m.dirsMu.RUnlock()

	if dir, exists := m.dirs[sessionID]; exists {
		return dir
	}
	return "home/guest" // default
}

func (m *mockSessionManager) SetCurrentDir(sessionID, dir string) {
	m.dirsMu.Lock()
	defer m.dirsMu.Unlock()
	m.dirs[sessionID] = dir
}
