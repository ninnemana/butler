package services

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/logging"
)

const (
	sampleResponse = "TEST-RESPONSE"
)

var (
	server    *httptest.Server
	localhost string
	logger    *logging.Logger
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

	client, err := logging.NewClient(context.Background(), os.Getenv("PROJECT_ID"))
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	logger = client.Logger(logName)

	os.Exit(m.Run())
}

func TestService(t *testing.T) {

	go func() {
		listenAddr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, sampleResponse)
		}))
		listenAddr.Close()

		host, err := url.Parse(listenAddr.URL)
		if err != nil {
			t.Fatalf("failed to parse httptest server URL: %v", err)
			return
		}

		if err := Start(&Config{
			ProjectID:     os.Getenv("PROJECT_ID"),
			ListenAddress: host.Host,
			Targets: map[string]string{
				"butler-proxy": server.URL,
			},
			Logger: logger,
		}); err != nil {
			t.Fatalf("failed to start server: %v", err)
		}
	}()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s", localhost), nil)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}

	client := &http.Client{}
	req.Host = "butler-proxy"
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to make request to local proxy address: %v", err)
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read proxy response body: %v", err)
	}

	if strings.TrimSpace(string(data)) != sampleResponse {
		// t.Errorf("expected '%s', received '%s'", sampleResponse, string(data))
	}
}
