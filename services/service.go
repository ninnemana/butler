package services

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
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
}

func Start(cfg Config) error {

	var err error
	var source net.Listener
	switch {
	case cfg.TLS == nil:
		source, err = net.Listen("tcp", cfg.LocalAddress)
	default:
		var cert tls.Certificate
		cert, err = tls.X509KeyPair(cfg.TLS.CertBlock, cfg.TLS.KeyBlock)
		if err != nil {
			return err
		}

		source, err = tls.Listen("tcp", cfg.LocalAddress, &tls.Config{
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
		},
		)
	}

	if err != nil {
		return err
	}

	// accept connections for the local address
	for {
		req, err := source.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %v\n", err)
			continue
		}

		fmt.Println(req.RemoteAddr(), req.LocalAddr())
		conn := req.(*net.TCPConn)

		r := &http.Request{}
		r.Write(conn)
		fmt.Println(r)
		conn.Close()
		req.Close()
		// handle the request
		// go handle(req)
	}

	return nil
}

func (s *Service) Close() {
	s.source.Close()
}

func (s *Service) handle(req net.Conn) {
	defer req.Close()

	var err error
	s.dest, err = net.Dial("tcp", s.ServiceAddress)
	if err != nil {
		log.Printf("failed to connect to service: %v\n", err)
		return
	}
	defer s.dest.Close()

	conn := req.(*net.TCPConn)
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(time.Second * 45)

	// pipe the request from the source connection to the destination
	go pipe(conn, s.dest.(*net.TCPConn))

	// pipe the response from the destination into the response of the source
	pipe(s.dest.(*net.TCPConn), conn)
}

func pipe(source *net.TCPConn, dest *net.TCPConn) {
	defer func() {
		dest.CloseWrite()
		source.CloseRead()
	}()
	io.Copy(dest, source)
}
