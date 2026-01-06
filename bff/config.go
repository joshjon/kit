package bff

import (
	"github.com/cohesivestack/valgo"

	"github.com/joshjon/kit/auth"
	"github.com/joshjon/kit/valgoutil"
)

type RegisterConfig struct {
	DownstreamURL string             `yaml:"downstreamURL" env:"DOWNSTREAM_URL"`
	OIDCProvider  OIDCProviderConfig `yaml:"oidcProvider" envPrefix:"OIDC_PROVIDER_"`
}

func (c *RegisterConfig) InitDefaults() {}

func (c *RegisterConfig) Validation() *valgo.Validation {
	v := valgo.New()
	v.Is(valgoutil.URLValidator(c.DownstreamURL, "downstreamURL"))
	v.In("oidcProvider", c.OIDCProvider.Validation())
	return v
}

type OIDCProviderConfig struct {
	SessionName string                         `yaml:"sessionName" env:"SESSION_NAME"`
	Endpoint    string                         `yaml:"endpoint" env:"ENDPOINT"`
	AppID       string                         `yaml:"appId" env:"APP_ID"`
	AppSecret   string                         `yaml:"appSecret" env:"APP_SECRET"`
	Redirects   auth.OIDCHandlerRedirectConfig `yaml:"redirects" envPrefix:"REDIRECTS_"`
	Audiences   []OIDCProviderAudienceScopes   `yaml:"audiences" envPrefix:"AUDIENCES_"`
}

func (c *OIDCProviderConfig) Validation() *valgo.Validation {
	v := valgo.New()
	v.Is(
		valgo.String(c.Endpoint, "endpoint").Not().Blank(),
		valgo.String(c.AppID, "appId").Not().Blank(),
		valgo.String(c.AppSecret, "appSecret").Not().Blank(),
	)
	v.In("redirects", c.Redirects.Validation())
	for i, aud := range c.Audiences {
		v.InRow("audiences", i, aud.Validation())
	}
	return v
}

type OIDCProviderAudienceScopes struct {
	Name   string   `yaml:"name" env:"NAME"`
	Path   string   `yaml:"path" env:"PATH"`
	Scopes []string `yaml:"scopes" env:"SCOPES"`
}

func (c *OIDCProviderAudienceScopes) Validation() *valgo.Validation {
	v := valgo.New()
	v.Is(
		valgoutil.URLValidator(c.Name, "name"),
		valgo.String(c.Path, "path").Not().Blank(),
	)
	for i, scope := range c.Scopes {
		v.InRow("scopes", i, valgo.Is(valgo.String(scope, "scope").Not().Blank()))
	}
	return v
}
