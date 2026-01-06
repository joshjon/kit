package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
)

func NewMemStore(secret string) sessions.Store {
	return memstore.NewStore([]byte(secret))
}
