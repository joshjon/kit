package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/context"
	"github.com/labstack/echo/v4"
)

func BearerTokenMiddleware(audPaths map[string]string, skipPathPrefixes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqPath := c.Request().URL.Path
			for _, prefix := range skipPathPrefixes {
				if strings.HasPrefix(reqPath, prefix) {
					return next(c)
				}
			}

			var resource string
			for aud, path := range audPaths {
				if strings.HasPrefix(reqPath, path) {
					resource = aud
					break
				}
			}

			p, err := GetOIDCProvider(c)
			if err != nil {
				return err
			}
			tkn, err := p.GetAccessToken(resource)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
			}

			defer context.Clear(c.Request())
			c.Request().Header.Set("Authorization", fmt.Sprintf("Bearer %s", tkn.Token))
			return next(c)
		}
	}
}
