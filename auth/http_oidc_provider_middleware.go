package auth

import (
	"errors"
	"net/http"

	"github.com/cohesivestack/valgo"
	"github.com/gin-contrib/sessions"
	"github.com/labstack/echo/v4"

	"github.com/joshjon/kit/valgoutil"
)

const oidcProviderContextKey = "auth-oidc-provider"

type OIDCProviderAudience struct {
	Name   string   `yaml:"name" env:"NAME"`
	Path   string   `yaml:"path" env:"PATH"`
	Scopes []string `yaml:"scopes" env:"SCOPES"`
}

func (c *OIDCProviderAudience) Validation() *valgo.Validation {
	v := valgo.New()
	v.Is(
		valgoutil.URLValidator(c.Name, "resource"),
		valgo.String(c.Path, "path").Not().Blank(),
	)
	for i, scope := range c.Scopes {
		v.InRow("scopes", i, valgo.Is(valgo.String(scope, "scope").Not().Blank()))
	}
	return v
}

type AccessToken struct {
	Token     string `json:"token"`
	Scope     string `json:"scope"`
	ExpiresAt int64  `json:"expiresAt"`
}

type OIDCProvider interface {
	SignInWithRedirectUri(redirectUri string) (string, error)
	HandleSignInCallback(request *http.Request) error
	SignOut(postLogoutRedirectUri string) (string, error)
	GetAccessToken(resource string) (AccessToken, error)
}

type OIDCProviderInitializer func(storage *SessionStorage) OIDCProvider

type OIDCProviderConfig struct {
	SessionName     string
	SessionStore    sessions.Store
	OIDCInitializer OIDCProviderInitializer
}

func OIDCProviderMiddleware(cfg OIDCProviderConfig, opts ...SessionStorageOption) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			s := &session{cfg.SessionName, c.Request(), cfg.SessionStore, nil, false, c.Response().Writer}
			p := cfg.OIDCInitializer(NewSessionStorage(s, opts...))
			c.Set(oidcProviderContextKey, p)
			return next(c)
		}
	}
}

func GetOIDCProvider(c echo.Context) (OIDCProvider, error) {
	v := c.Get(oidcProviderContextKey)
	if v == nil {
		return nil, errors.New("oidc provider not found")
	}
	p, ok := v.(OIDCProvider)
	if !ok {
		return nil, errors.New("found an invalid oidc provider value")
	}
	return p, nil
}
