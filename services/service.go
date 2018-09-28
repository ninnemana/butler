package services

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
)

var (
	hostTarget = map[string]string{}
	hostProxy  = map[string]*httputil.ReverseProxy{}
)

type Service struct {
	LocalAddress   string
	ServiceAddress string
	Log            bool

	source net.Listener
	dest   net.Conn
}

type Config struct {
	LocalAddress string
	TLS          *TLS
}

type TLS struct {
	CertBlock []byte
	KeyBlock  []byte

	CertFile string
	KeyFile  string
}

func Start(cfg Config) error {

	var tlsConfig *tls.Config
	if cfg.TLS != nil {
		var cert tls.Certificate
		var err error

		switch {
		case cfg.TLS.CertBlock != nil && cfg.TLS.KeyBlock != nil:
			cert, err = tls.X509KeyPair(cfg.TLS.CertBlock, cfg.TLS.KeyBlock)
		case cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "":
			cert, err = tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		}

		if err != nil {
			return err
		}

		tlsConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			Certificates: []tls.Certificate{cert},
		}
	}

	h := &handler{}
	http.Handle("/", h)
	server := &http.Server{
		Addr:      cfg.LocalAddress,
		Handler:   h,
		TLSConfig: tlsConfig,
	}

	return server.ListenAndServe()
}
