package auth

import (
	"net/http"
	"time"

	"github.com/cohesivestack/valgo"
	"github.com/labstack/echo/v4"

	"github.com/joshjon/kit/valgoutil"
)

type OIDCHandlerRedirectConfig struct {
	BaseAuthServerURI     string `yaml:"baseAuthServerURI" env:"BASE_AUTH_SERVER_URI"`
	PostLogoutRedirectURI string `yaml:"postLogoutRedirectURI" env:"POST_LOGOUT_REDIRECT_URI"`
}

func (c *OIDCHandlerRedirectConfig) Validation() *valgo.Validation {
	return valgo.Is(
		valgoutil.URLValidator(c.BaseAuthServerURI, "baseAuthServerURI"),
		valgoutil.URLValidator(c.PostLogoutRedirectURI, "postLogoutRedirectURI"),
	)
}

type OIDCHandler struct {
	sessionName string
	redirects   OIDCHandlerRedirectConfig
	pathPrefix  string
}

func NewOIDCHandler(sessionName string, registeredPathPrefix string, redirectCfg OIDCHandlerRedirectConfig) *OIDCHandler {
	return &OIDCHandler{
		sessionName: sessionName,
		pathPrefix:  registeredPathPrefix,
		redirects:   redirectCfg,
	}
}

func (h *OIDCHandler) Register(g *echo.Group) {
	g.GET("/login", h.Login)
	g.GET("/callback", h.LoginCallback)
	g.GET("/logout", h.Logout)
}

func (h *OIDCHandler) Login(c echo.Context) error {
	p, err := GetOIDCProvider(c)
	if err != nil {
		return err
	}
	signInURI, err := p.SignInWithRedirectUri(h.redirects.BaseAuthServerURI + h.pathPrefix + "/callback")
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusTemporaryRedirect, signInURI)
}

func (h *OIDCHandler) LoginCallback(c echo.Context) error {
	p, err := GetOIDCProvider(c)
	if err != nil {
		return err
	}
	if err = p.HandleSignInCallback(c.Request()); err != nil {
		return err
	}
	return c.Redirect(http.StatusTemporaryRedirect, h.redirects.PostLogoutRedirectURI)
}

func (h *OIDCHandler) Logout(c echo.Context) error {
	p, err := GetOIDCProvider(c)
	if err != nil {
		return err
	}
	signOutUri, signOutErr := p.SignOut(h.redirects.PostLogoutRedirectURI)
	if signOutErr != nil {
		return c.String(http.StatusOK, signOutErr.Error())
	}

	c.SetCookie(&http.Cookie{
		Name:     h.sessionName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // expire the existing cookie
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.Redirect(http.StatusTemporaryRedirect, signOutUri)
}
