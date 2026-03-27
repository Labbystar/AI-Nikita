package bot

import "sync"

type SessionState struct {
	PendingAction  string
	DraftSource    string
	DraftMeetingID string
	AwaitingUpload bool
}

type StateStore struct {
	mu    sync.RWMutex
	store map[int64]*SessionState
}

func NewStateStore() *StateStore {
	return &StateStore{store: map[int64]*SessionState{}}
}

func (s *StateStore) Get(chatID int64) *SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.store[chatID]
	if !ok {
		st = &SessionState{}
		s.store[chatID] = st
	}
	return st
}

func (s *StateStore) Reset(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[chatID] = &SessionState{}
}
