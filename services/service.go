package services

import (
	"crypto/tls"
	"net/http"

	"github.com/pkg/errors"
)

type Config struct {
	ListenAddress string            `json:"listenAddress,omitempty"`
	TLS           *TLS              `json:"tls,omitempty"`
	Targets       map[string]string `json:"targets,omitempty"`
}

type TLS struct {
	Enforce   bool   `json:"enforce,omitempty"`
	CertBlock []byte `json:"cert_block,omitempty"`
	KeyBlock  []byte `json:"key_block,omitempty"`

	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"key_file,omitempty"`
}

func Start(cfg Config) error {

	h := &handler{
		Targets: cfg.Targets,
	}
	http.Handle("/", h)

	if cfg.TLS == nil {

		server := &http.Server{
			Addr:    cfg.ListenAddress,
			Handler: h,
		}

		return errors.Wrap(
			server.ListenAndServe(),
			"fell out of listening for HTTP traffic",
		)
	}

	// tell the handler if it's supposed to
	// enforce HTTPS usage.
	h.EnforceSSL = cfg.TLS.Enforce

	var cert tls.Certificate
	var err error

	switch {
	case cfg.TLS.CertBlock != nil && cfg.TLS.KeyBlock != nil:
		cert, err = tls.X509KeyPair(cfg.TLS.CertBlock, cfg.TLS.KeyBlock)
	case cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "":
		cert, err = tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
	}

	if err != nil {
		return errors.Wrap(err, "failed to read TLS certificate")
	}

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		Certificates: []tls.Certificate{cert},
	}

	tlsChan := make(chan error)
	unsecure := make(chan error)
	go func() {
		srv := &http.Server{
			Addr:      ":443",
			Handler:   h,
			TLSConfig: tlsConfig,
		}
		tlsChan <- srv.ListenAndServeTLS("", "")
	}()

	go func() {
		srv := &http.Server{
			Addr:    cfg.ListenAddress,
			Handler: h,
		}

		unsecure <- errors.Wrap(
			srv.ListenAndServe(),
			"failed to listen for HTTP traffic",
		)
	}()

	select {
	case err := <-tlsChan:
		return err
	case err := <-unsecure:
		return err
	}
}
