package bff

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/labstack/echo/v4"

	"github.com/joshjon/kit/auth"
	"github.com/joshjon/kit/proxy"
	"github.com/joshjon/kit/server"
)

type Registerer interface {
	Register(pathPrefix string, h server.Handler, middleware ...echo.MiddlewareFunc)
}

func RegisterAuthHandler(cfg OIDCProviderConfig, srv Registerer, sessionName string, middleware ...echo.MiddlewareFunc) {
	srv.Register("/auth", auth.NewOIDCHandler(sessionName, "/auth", cfg.Redirects), middleware...)
}

func RegisterReverseProxyHandler(srv Registerer, client *http.Client, downstreamURL string, pathPrefixes []string, middleware ...echo.MiddlewareFunc) {
	for _, pathPrefix := range pathPrefixes {
		srv.Register(pathPrefix, proxy.NewReverseProxyHandler(client, downstreamURL), middleware...)
	}
}

func NewMiddleware(
	audScopes []OIDCProviderAudienceScopes,
	provInit auth.OIDCProviderInitializer,
	sessionName string,
	sessionStore sessions.Store,
	sessionStorageOpts ...auth.SessionStorageOption,
) []echo.MiddlewareFunc {
	audPaths := map[string]string{}
	for _, aud := range audScopes {
		audPaths[aud.Name] = aud.Path
	}

	return []echo.MiddlewareFunc{
		auth.OIDCProviderMiddleware(auth.OIDCProviderConfig{
			SessionName:     sessionName,
			SessionStore:    sessionStore,
			OIDCInitializer: provInit,
		}, sessionStorageOpts...),
		auth.BearerTokenMiddleware(audPaths, "/healthz", "/auth"),
	}
}
