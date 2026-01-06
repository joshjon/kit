package bff

import (
	"github.com/logto-io/go/v2/client"

	"github.com/joshjon/kit/auth"
	"github.com/joshjon/kit/logto"
)

func NewLogtoOIDCProviderInitializer(oidcCfg OIDCProviderConfig) auth.OIDCProviderInitializer {
	ltCfg := &client.LogtoConfig{
		Endpoint:  oidcCfg.Endpoint,
		AppId:     oidcCfg.AppID,
		AppSecret: oidcCfg.AppSecret,
	}

	for _, aud := range oidcCfg.Audiences {
		ltCfg.Resources = append(ltCfg.Resources, aud.Name)
		ltCfg.Scopes = append(ltCfg.Scopes, aud.Scopes...)
	}

	return logto.OIDCProviderInitializer(ltCfg)
}
