package logto

import (
	"github.com/logto-io/go/v2/client"

	"github.com/joshjon/kit/auth"
)

func OIDCProviderInitializer(cfg *client.LogtoConfig) auth.OIDCProviderInitializer {
	return func(storage *auth.SessionStorage) auth.OIDCProvider {
		return NewClient(cfg, storage)
	}
}

var _ auth.OIDCProvider = (*Client)(nil)

type Client struct {
	*client.LogtoClient
	cfg *client.LogtoConfig
}

func NewClient(cfg *client.LogtoConfig, storage *auth.SessionStorage) *Client {
	return &Client{
		LogtoClient: client.NewLogtoClient(cfg, storage),
		cfg:         cfg,
	}
}

func (c *Client) GetAccessToken(resource string) (auth.AccessToken, error) {
	// Debug: Check if token is in session before calling Logto SDK
	// The Logto SDK should be checking session storage internally, but let's verify
	tkn, err := c.LogtoClient.GetAccessToken(resource)
	if err != nil {
		return auth.AccessToken{}, err
	}
	// TODO: Add logging to see if this is fetching from cache or making network call
	return auth.AccessToken(tkn), nil
}
