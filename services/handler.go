package services

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type handler struct {
	Targets map[string]string
	proxies map[string]*httputil.ReverseProxy
	l       sync.Mutex
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodConnect {
		tunnel(w, r)
		return
	}

	host := r.Host

	target, ok := h.Targets[host]
	if !ok {
		notFound(w)
		return
	}

	remote, err := url.Parse(target)
	if err != nil {
		log.Println("target parse fail:", err)
		return
	}

	r.Host = remote.Host

	if fn, ok := h.proxies[host]; ok {
		fn.ServeHTTP(w, r)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)

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

	proxy.ServeHTTP(w, r)
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

func tunnel(w http.ResponseWriter, r *http.Request) {
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

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
