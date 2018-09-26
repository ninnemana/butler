package services

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

const (
	sampleResponse = "TEST RESPONSE"
)

var (
	server    *httptest.Server
	localhost string
)

func TestMain(m *testing.M) {
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, sampleResponse)
	}))

	host, err := url.Parse(server.URL)
	if err != nil {
		log.Fatalf("failed to parse httptest server URL: %v", err)
		return
	}

	localhost = host.Host

	os.Exit(m.Run())
}

func TestService(t *testing.T) {

	svc := &Service{
		LocalAddress:   "localhost:8161",
		ServiceAddress: localhost,
	}
	Start(svc)
	defer func() {
		if recover() != nil {
			svc.Close()
		}
	}()

	res, err := http.Get("http://localhost:8161")
	if err != nil {
		t.Fatalf("failed to make request to local proxy address: %v", err)
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read proxy response body: %v", err)
	}

	if strings.TrimSpace(string(data)) != sampleResponse {
		t.Fatalf("expected '%s', received '%s'", sampleResponse, string(data))
	}
}
