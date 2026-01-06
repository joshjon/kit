package bff

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/labstack/echo/v4"
	"github.com/logto-io/go/v2/client"

	"github.com/joshjon/kit/auth"
	"github.com/joshjon/kit/logto"
	"github.com/joshjon/kit/proxy"
	"github.com/joshjon/kit/server"
)

type Registerer interface {
	Register(pathPrefix string, h server.Handler, middleware ...echo.MiddlewareFunc)
}

func Register(cfg RegisterConfig, srv Registerer, client *http.Client, sessionStore sessions.Store, pathPrefixes ...string) error {
	proxyURLs := []string{cfg.DownstreamURL}
	for _, proxyURL := range proxyURLs {
		if err := waitDownstreamHealthy(client, proxyURL); err != nil {
			return err
		}
	}

	srv.Register("/auth", auth.NewOIDCHandler(cfg.OIDCProvider.SessionName, "/auth", cfg.OIDCProvider.Redirects))

	for _, pathPrefix := range pathPrefixes {
		srv.Register(pathPrefix, proxy.NewReverseProxyHandler(client, cfg.DownstreamURL), NewMiddlewares(cfg.OIDCProvider, sessionStore)...)
	}

	return nil
}

func NewMiddlewares(oidcCfg OIDCProviderConfig, sessionStore sessions.Store) []echo.MiddlewareFunc {
	ltCfg := &client.LogtoConfig{
		Endpoint:  oidcCfg.Endpoint,
		AppId:     oidcCfg.AppID,
		AppSecret: oidcCfg.AppSecret,
	}

	audPaths := map[string]string{}

	for _, aud := range oidcCfg.Audiences {
		ltCfg.Resources = append(ltCfg.Resources, aud.Name)
		ltCfg.Scopes = append(ltCfg.Scopes, aud.Scopes...)
		audPaths[aud.Name] = aud.Path
	}

	logtoInit := logto.OIDCProviderInitializer(ltCfg)

	return []echo.MiddlewareFunc{
		auth.OIDCProviderMiddleware(auth.OIDCProviderConfig{
			SessionName:     oidcCfg.SessionName,
			SessionStore:    sessionStore,
			OIDCInitializer: logtoInit,
		}),
		auth.BearerTokenMiddleware(audPaths, "/healthz", "/auth"),
	}
}
