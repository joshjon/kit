package jwt

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/cohesivestack/valgo"
	"github.com/labstack/echo/v4"

	"github.com/joshjon/kit/errtag"
	"github.com/joshjon/kit/valgoutil"
)

const (
	authUserIDContextKey = "jwt-auth-user-id"
	authEmailContextKey  = "jwt-auth-email"
)

type PathScopesConfig struct {
	Prefix       string              `yaml:"prefix" env:"PREFIX"`
	MethodScopes map[string][]string `yaml:"scopes" env:"SCOPES"`
}

func (p *PathScopesConfig) Validation() *valgo.Validation {
	v := valgo.New()
	v.Is(valgo.String(p.Prefix, "prefix").Not().Blank())
	for method, scopes := range p.MethodScopes {
		v.In("scopes", valgo.Is(valgo.String(method, "method").Not().Blank()))
		for i, scope := range scopes {
			v.InRow("scopes."+method, i, valgo.Is(valgo.String(scope, "scope").Not().Blank()))
		}
	}
	return v
}

type AudienceConfig struct {
	Name  string             `yaml:"name" env:"AUDIENCE"`
	Paths []PathScopesConfig `yaml:"paths" envPrefix:"PATHS_"`
}

func (a *AudienceConfig) Validation() *valgo.Validation {
	v := valgo.New()
	v.Is(valgoutil.URLValidator(a.Name, "name"))
	for i, path := range a.Paths {
		v.InRow("paths", i, path.Validation())
	}
	return v
}

type Config struct {
	IssuerURL            string                       `yaml:"issuerURL" env:"ISSUER_URL"`
	Audiences            []AudienceConfig             `yaml:"audiences" envPrefix:"AUDIENCES_"`
	SignatureAlgorithm   validator.SignatureAlgorithm `yaml:"signatureAlgorithm" env:"SIGNATURE_ALGORITHM"`
	CacheDurationSeconds int                          `yaml:"cacheDurationSeconds" env:"CACHE_DURATION_SECONDS"`
}

func (c *Config) InitDefaults() {
	c.CacheDurationSeconds = int((10 * time.Minute).Seconds())
}

func (c *Config) Validation() *valgo.Validation {
	v := valgo.Is(
		valgoutil.URLValidator(c.IssuerURL, "issuerURL"),
		valgo.Int(c.CacheDurationSeconds, "cacheDurationSeconds").GreaterOrEqualTo(0),
		valgo.String(c.SignatureAlgorithm, "signatureAlgorithm").Not().Blank(),
	)
	for i, aud := range c.Audiences {
		v.InRow("audiences", i, aud.Validation())
	}
	return v
}

func ValidateMiddleware(cfg Config, skipNonMatchingPrefix bool, skipPathPrefixes ...string) (echo.MiddlewareFunc, error) {
	issuerURL, err := url.Parse(cfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	cacheTTL := time.Second * time.Duration(cfg.CacheDurationSeconds)
	provider := jwks.NewCachingProvider(issuerURL, cacheTTL)

	type audScopes struct {
		aud          string
		methodScopes map[string][]string
	}

	pathAudScopes := map[string]audScopes{}

	for _, aud := range cfg.Audiences {
		for _, path := range aud.Paths {
			pathAudScopes[path.Prefix] = audScopes{
				aud:          aud.Name,
				methodScopes: path.MethodScopes,
			}
		}
	}

	// support additive path prefixes by finding the longest prefix match
	getAudAndScopes := func(c echo.Context) ([]string, []string, bool) {
		reqPath := c.Request().URL.Path
		var longestPrefixMatch string
		for prefix := range pathAudScopes {
			if strings.HasPrefix(reqPath, prefix) {
				if len(prefix) > len(longestPrefixMatch) {
					longestPrefixMatch = prefix
				}
			}
		}
		if longestPrefixMatch == "" {
			return nil, nil, false // no matching prefix found in config
		}

		match := pathAudScopes[longestPrefixMatch]

		return []string{match.aud}, match.methodScopes[c.Request().Method], true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqPath := c.Request().URL.Path
			for _, prefix := range skipPathPrefixes {
				if strings.HasPrefix(reqPath, prefix) {
					return next(c)
				}
			}

			bearer := c.Request().Header.Get("Authorization")
			if bearer == "" {
				return errtag.NewTagged[errtag.Unauthorized]("authorization header not found")
			}
			if !strings.HasPrefix(bearer, "Bearer ") {
				return errtag.NewTagged[errtag.Unauthorized]("authorization header must start with 'Bearer '")
			}

			token := strings.TrimPrefix(bearer, "Bearer ")

			aud, scopes, ok := getAudAndScopes(c)
			if !ok && skipNonMatchingPrefix {
				return next(c)
			}

			jwtValidator, err := validator.New(
				provider.KeyFunc,
				cfg.SignatureAlgorithm,
				issuerURL.String(),
				aud,
				validator.WithCustomClaims(func() validator.CustomClaims {
					return &Claims{
						requiredScopes: scopes,
					}
				}),
			)
			if err != nil {
				return fmt.Errorf("create jwt validator: %w", err)
			}

			claims, err := jwtValidator.ValidateToken(c.Request().Context(), token)
			if err != nil {
				return err
			}

			validated, ok := claims.(*validator.ValidatedClaims)
			if !ok {
				return errtag.NewTagged[errtag.Unauthorized]("invalid claims type")
			}
			if customClaims, ok := validated.CustomClaims.(*Claims); ok {
				c.Set(authEmailContextKey, customClaims.Email)
			}

			c.Set(authUserIDContextKey, validated.RegisteredClaims.Subject)

			return next(c)
		}
	}, nil
}

type Claims struct {
	Scope          string `json:"scope"`
	Email          string `json:"email"`
	requiredScopes []string
}

func (s *Claims) Validate(context.Context) error {
	scopes := map[string]struct{}{}
	for _, scope := range strings.Split(s.Scope, " ") {
		scopes[scope] = struct{}{}
	}
	for _, required := range s.requiredScopes {
		if _, ok := scopes[required]; !ok {
			return errtag.NewTagged[errtag.Unauthorized]("required scope not found in claims")
		}
	}
	if s.Email == "" {
		return errtag.NewTagged[errtag.Unauthorized]("email not found in claims")
	}
	return nil
}

func AuthUserIDFromContext(c echo.Context) (string, error) {
	s, ok := c.Get(authUserIDContextKey).(string)
	if !ok {
		return "", errtag.NewTagged[errtag.Unauthorized]("auth user id not found in context")
	}
	return s, nil
}

func EmailFromContext(c echo.Context) (string, error) {
	s, ok := c.Get(authEmailContextKey).(string)
	if !ok {
		return "", errtag.NewTagged[errtag.Unauthorized]("auth email not found in context")
	}
	return s, nil
}
