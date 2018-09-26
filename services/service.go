package services

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"time"

	"github.com/pkg/errors"
)

type Service struct {
	LocalAddress   string
	ServiceAddress string
	Log            bool
	TLS            *tls.Config

	source net.Listener
	dest   net.Conn
}

func Start(svcs ...*Service) {
	ch := make(chan bool)
	for _, svc := range svcs {
		go func(s *Service) {
			if err := s.Listen(); err != nil {
				log.Fatalf("fell out of service listener: %v", err)
			}
		}(svc)
	}
	<-ch
}

func (s *Service) Close() {
	s.source.Close()
}

func (s *Service) Listen() error {
	var err error

	// open TCP listener for source
	switch {
	case s.TLS == nil:
		s.source, err = net.Listen("tcp", s.LocalAddress)
	default:
		s.source, err = tls.Listen("tcp", s.LocalAddress, s.TLS)
	}

	if err != nil {
		return errors.Wrap(err, "failed to create TCP listener")
	}

	// accept connections for the local address
	for {
		req, err := s.source.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %v\n", err)
			continue
		}

		// handle the request
		go s.handle(req)
	}
}

func (s *Service) handle(req net.Conn) {
	defer req.Close()

	var err error
	// dial a connection with the service
	switch {
	case s.TLS == nil:
		s.dest, err = net.Dial("tcp", s.ServiceAddress)
	default:
		s.dest, err = tls.Dial("tcp", s.ServiceAddress, s.TLS)
	}

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
