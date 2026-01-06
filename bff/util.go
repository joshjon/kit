package bff

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joshjon/kit/log"
	"github.com/joshjon/kit/server"
)

const clientTimeout = 60 * time.Second

type httpTLSConfig struct {
	certFile   string
	keyFile    string
	caCertFile string
}

func createHTTPClient(tlsCfg *httpTLSConfig) (*http.Client, error) {
	client := http.DefaultClient
	client.Timeout = clientTimeout

	if tlsCfg == nil {
		return client, nil
	}

	cert, err := tls.LoadX509KeyPair(tlsCfg.certFile, tlsCfg.keyFile)
	if err != nil {
		return nil, err
	}

	caCert, err := os.ReadFile(tlsCfg.caCertFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("failed to append ca cert")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client.Transport = transport
	return client, nil
}

func waitDownstreamHealthy(client *http.Client, addr string) error {
	healthzURL := fmt.Sprintf("%s/healthz", addr)
	maxRetries := 15
	interval := time.Second

	var res *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		res, err = client.Get(healthzURL)
		if err == nil && res.StatusCode == http.StatusOK {
			return nil
		}

		time.Sleep(interval)
	}

	if err != nil {
		return fmt.Errorf("downstream unhealthy: %w", err)
	} else if res != nil {
		return fmt.Errorf("downstream unhealthy: %s", http.StatusText(res.StatusCode))
	}

	return errors.New("downstream unhealthy")
}

func serve(ctx context.Context, srv *server.Server, logger log.Logger) error {
	errs := make(chan error)

	logger.Info("starting server", "address", srv.Address())
	go func() {
		defer close(errs)
		if err := srv.Start(); err != nil {
			errs <- fmt.Errorf("start server: %w", err)
		}
	}()
	defer srv.Stop(ctx) //nolint:errcheck

	logger.Info("waiting for server to be healthy")
	if err := srv.WaitHealthy(15, time.Second); err != nil {
		return err
	}
	logger.Info("server healthy")

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		logger.Info("server stopped")
		return nil
	}
}
