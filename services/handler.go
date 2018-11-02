package services

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var (
	onExitFlushLoop func()

	// Hop-by-hop headers. These are removed when sent to the backend.
	// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
	hopHeaders = []string{
		"Connection",
		"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",      // canonicalized version of "TE"
		"Trailer", // not Trailers per URL above; http://www.rfc-editor.org/errata_search.php?eid=4522
		"Transfer-Encoding",
		"Upgrade",
	}
)

type requestCanceler interface {
	CancelRequest(req *http.Request)
}

type handler struct {
	targets map[string]string
	proxies map[string]*httputil.ReverseProxy
	l       sync.Mutex
}

func NewHandler() *handler {
	return &handler{
		targets: map[string]string{},
		proxies: map[string]*httputil.ReverseProxy{},
	}
}

func (h *handler) PutTarget(src, dest string) error {
	if cur, ok := h.targets[src]; ok {
		return errors.Errorf("route '%s' is already assigned to '%s'", src, cur)
	}

	h.l.Lock()
	h.targets[src] = dest
	h.l.Unlock()

	return nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodConnect:
		h.tunnel(w, r)
	default:
		h.proxy(w, r)
	}
}

func (h *handler) proxy(w http.ResponseWriter, r *http.Request) {
	// host := r.Host

	target, ok := h.targets[r.Host]
	if !ok {
		notFound(w)
		return
	}

	remote, err := url.Parse(target)
	if err != nil {
		log.Println("target parse fail:", err)
		return
	}

	transport := http.DefaultTransport

	outreq := new(http.Request)
	*outreq = *r

	if cn, ok := w.(http.CloseNotifier); ok {
		if canceler, ok := transport.(requestCanceler); ok {
			done := make(chan struct{})
			defer close(done)
			gone := cn.CloseNotify()

			go func() {
				select {
				case <-gone:
					canceler.CancelRequest(outreq)
				case <-done:
				}
			}()
		}
	}

	director(remote, r)
	outreq.Close = false

	outreq.Header = make(http.Header)
	for k, vv := range r.Header {
		for _, v := range vv {
			outreq.Header.Add(k, v)
		}
	}

	removeHeaders(outreq.Header)

	addXForwardedForHeader(outreq)
	res, err := transport.RoundTrip(outreq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	removeHeaders(res.Header)

	copyHeader(w.Header(), res.Header)
	if len(res.Trailer) > 0 {
		trailerKeys := make([]string, 0, len(res.Trailer))
		for k := range res.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		w.Header().Add("Trailer", strings.Join(trailerKeys, ", "))
	}

	w.WriteHeader(res.StatusCode)
	if len(res.Trailer) > 0 {
		// Force chunking if we saw a response trailer.
		// This prevents net/http from calculating the length for short
		// bodies and adding a Content-Length.
		if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
	}

	io.Copy(w, res.Body)
	res.Body.Close()
	copyHeader(w.Header(), res.Trailer)
}

func (h *handler) tunnel(w http.ResponseWriter, r *http.Request) {
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func addXForwardedForHeader(req *http.Request) {
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := req.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		req.Header.Set("X-Forwarded-For", clientIP)
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func director(target *url.URL, r *http.Request) {
	r.URL.Scheme = target.Scheme
	r.URL.Host = target.Host
	r.URL.Path = singleJoiningSlash(target.Path, r.URL.Path)

	r.Host = r.URL.Host
	switch {
	case target.RawQuery == "" || r.URL.RawQuery == "":
		r.URL.RawQuery = target.RawQuery + r.URL.RawQuery
	default:
		r.URL.RawQuery = target.RawQuery + "&" + r.URL.RawQuery
	}

	if _, ok := r.Header["User-Agent"]; !ok {
		r.Header.Set("User-Agent", "")
	}
}

func notFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)

	res, err := http.Get("https://i.pinimg.com/736x/70/01/aa/7001aa5d1483fa70cd00022ed226483d.jpg")
	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	w.Write(data)
}

func removeHeaders(header http.Header) {
	// Remove hop-by-hop headers listed in the "Connection" header.
	if c := header.Get("Connection"); c != "" {
		for _, f := range strings.Split(c, ",") {
			if f = strings.TrimSpace(f); f != "" {
				header.Del(f)
			}
		}
	}

	// Remove hop-by-hop headers
	for _, h := range hopHeaders {
		if header.Get(h) != "" {
			header.Del(h)
		}
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
