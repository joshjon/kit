package auth

import (
	"github.com/gin-contrib/sessions"
)

type SessionStorage struct {
	session    sessions.Session
	errHandler func(err error)
}

func NewSessionStorage(session sessions.Session) *SessionStorage {
	return &SessionStorage{
		session: session,
	}
}

func (s *SessionStorage) GetItem(key string) string {
	value := s.session.Get(key)
	if value == nil {
		return ""
	}
	return value.(string)
}

func (s *SessionStorage) SetItem(key, value string) {
	s.session.Set(key, value)
	if err := s.session.Save(); err != nil && s.errHandler != nil {
		s.errHandler(err)
	}
}
