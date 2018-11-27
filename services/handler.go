package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"cloud.google.com/go/logging"
	"go.opencensus.io/trace"
)

type handler struct {
	EnforceSSL bool
	targets    map[string]string
	logger     *logging.Logger
	proxies    map[string]*httputil.ReverseProxy
	projectID  string
	l          sync.Mutex
}

type request struct {
	entry    logging.Entry
	span     *trace.Span
	response http.ResponseWriter
	request  *http.Request
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(context.Background(), "butler")
	defer span.End()

	span.AddAttributes(trace.StringAttribute("http.host", r.Host))
	span.AddAttributes(trace.StringAttribute("http.method", r.Method))
	span.AddAttributes(trace.StringAttribute("http.path", r.URL.Path))
	span.AddAttributes(trace.StringAttribute("http.route", r.URL.Path))
	span.AddAttributes(trace.StringAttribute("http.user_agent", r.UserAgent()))
	span.AddAttributes(trace.StringAttribute("http.url", r.URL.String()))

	req := &request{
		request:  r.WithContext(trace.NewContext(r.Context(), span)),
		response: w,
		span:     span,
		entry: logging.Entry{
			Timestamp: time.Now().UTC(),
			Severity:  logging.Info,
			Labels:    map[string]string{},
			HTTPRequest: &logging.HTTPRequest{
				Request: r,
				Status:  http.StatusTemporaryRedirect,
			},
			Payload: "Handling HTTP Request",
			Trace:   fmt.Sprintf("projects/%s/traces/%s", h.projectID, span.SpanContext().TraceID),
		},
	}
	h.logger.Log(req.entry)

	h.forceSSL(req)

	host := req.request.Host
	target, ok := h.targets[host]
	if !ok {
		h.notFound(req)
		return
	}

	remote, err := url.Parse(target)
	if err != nil {
		req.entry.Payload = err
		req.entry.Severity = logging.Error
		h.logger.Log(req.entry)
		return
	}

	req.request.Host = remote.Host

	if fn, ok := h.proxies[host]; ok {
		req.entry.Payload = "Redirecting to Service"
		req.entry.Labels["service"] = host
		fn.ServeHTTP(req.response, req.request)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}

	h.l.Lock()
	switch h.proxies {
	case nil:
		h.proxies = map[string]*httputil.ReverseProxy{
			host: proxy,
		}
	default:
		h.proxies[host] = proxy
	}
	h.l.Unlock()

	proxy.ServeHTTP(req.response, req.request)
}

func (h *handler) forceSSL(r *request) {
	if !h.EnforceSSL || r.request.TLS != nil {
		return
	}

	r.entry.Payload = "Redirecting to HTTP(s)"
	h.logger.Log(r.entry)

	redirect := fmt.Sprintf("https://%s%s", r.request.Host, r.request.URL.Path)
	if r.request.URL.RawQuery != "" {
		redirect = redirect + "?" + r.request.URL.RawQuery
	}
	http.Redirect(
		r.response,
		r.request,
		redirect,
		http.StatusTemporaryRedirect,
	)
}

func (h *handler) notFound(r *request) error {
	r.response.WriteHeader(http.StatusNotFound)

	res, err := http.Get("https://i.pinimg.com/736x/70/01/aa/7001aa5d1483fa70cd00022ed226483d.jpg")
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	_, err = r.response.Write(data)
	return err
}
