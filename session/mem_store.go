package session

import (
	"encoding/hex"
	"fmt"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
)

func NewMemStore(secret string) (sessions.Store, error) {
	b, err := hex.DecodeString(secret)
	if err != nil {
		return nil, fmt.Errorf("hex decode session secret: %w", err)
	}
	return memstore.NewStore(b), nil
}
