package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/labstack/echo/v4"
)

type ReverseProxyHandler struct {
	client *http.Client
	apiURL string
}

func NewReverseProxyHandler(client *http.Client, apiURL string) *ReverseProxyHandler {
	return &ReverseProxyHandler{
		client: client,
		apiURL: apiURL,
	}
}

func (h *ReverseProxyHandler) Register(g *echo.Group) {
	g.Any("/*", h.Handle)
}

func (h *ReverseProxyHandler) Handle(c echo.Context) error {
	targetURL, err := url.Parse(h.apiURL)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Bad target URL")
	}

	// Create a reverse proxy that directs requests to the downstream API
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = h.client.Transport
	proxy.ServeHTTP(c.Response().Writer, c.Request())
	return nil
}
