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

func RegisterAuthHandler(cfg OIDCProviderConfig, srv Registerer, sessionName string) {
	srv.Register("/auth", auth.NewOIDCHandler(sessionName, "/auth", cfg.Redirects))
}

func RegisterReverseProxyHandler(
	cfg OIDCProviderConfig,
	srv Registerer,
	client *http.Client,
	provInit auth.OIDCProviderInitializer,
	sessionStore sessions.Store,
	sessionName string,
	downstreamURL string,
	pathPrefixes ...string,
) error {
	proxyURLs := []string{downstreamURL}
	for _, proxyURL := range proxyURLs {
		if err := waitDownstreamHealthy(client, proxyURL); err != nil {
			return err
		}
	}

	for _, pathPrefix := range pathPrefixes {
		srv.Register(pathPrefix, proxy.NewReverseProxyHandler(client, downstreamURL), NewMiddlewares(cfg.Audiences, sessionName, provInit, sessionStore)...)
	}

	return nil
}

func NewMiddlewares(
	audScopes []OIDCProviderAudienceScopes,
	sessionName string,
	provInit auth.OIDCProviderInitializer,
	sessionStore sessions.Store,
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
		}),
		auth.BearerTokenMiddleware(audPaths, "/healthz", "/auth"),
	}
}
