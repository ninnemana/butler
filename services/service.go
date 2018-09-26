package services

import (
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

	source net.Listener
}

func pipe(source *net.TCPConn, dest *net.TCPConn) {
	defer dest.CloseWrite()
	defer source.CloseRead()
	io.Copy(dest, source)
}

func Start(svcs ...*Service) {
	for _, svc := range svcs {
		go func(s *Service) {
			if err := s.Listen(); err != nil {
				log.Fatalf("fell out of service listener: %v", err)
			}
		}(svc)
	}
}

func (s *Service) Close() {
	s.source.Close()
}

func (s *Service) Listen() error {
	var err error

	// open TCP listener for source
	s.source, err = net.Listen("tcp", s.LocalAddress)
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
		go handle(req, s.ServiceAddress)
	}
}

func handle(req net.Conn, svcAddr string) {
	defer req.Close()

	// dial a connection with the service
	dest, err := net.Dial("tcp", svcAddr)
	if err != nil {
		log.Printf("failed to connect to service: %v\n", err)
		return
	}
	defer dest.Close()

	conn := req.(*net.TCPConn)
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(time.Second * 45)

	// pipe the request from the source connection to the destination
	go pipe(conn, dest.(*net.TCPConn))

	// pipe the response from the destination into the response of the source
	pipe(dest.(*net.TCPConn), conn)
}
