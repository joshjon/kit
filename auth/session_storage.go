package auth

import (
	"fmt"

	"github.com/gin-contrib/sessions"
)

type SessionStorage struct {
	session    sessions.Session
	errHandler func(err error)
}

type SessionStorageOption func(*SessionStorage)

// WithErrHandler sets a custom error handler.
func WithErrHandler(handler func(err error)) SessionStorageOption {
	return func(s *SessionStorage) {
		s.errHandler = handler
	}
}

func NewSessionStorage(session sessions.Session, opts ...SessionStorageOption) *SessionStorage {
	s := &SessionStorage{
		session: session,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *SessionStorage) GetItem(key string) string {
	value := s.session.Get(key)
	if value == nil {
		return ""
	}
	return value.(string)
}

func (s *SessionStorage) SetItem(key string, value string) {
	s.session.Set(key, value)
	if err := s.session.Save(); err != nil {
		if s.errHandler != nil {
			s.errHandler(fmt.Errorf("token not cached: session save error: %w", err))
		}
	}
}
