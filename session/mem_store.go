package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
)

func NewMemStore(key []byte) (sessions.Store, error) {
	return memstore.NewStore(key), nil
}
