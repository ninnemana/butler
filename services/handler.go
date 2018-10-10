package services

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type handler struct {
	Targets map[string]string
	proxies map[string]*httputil.ReverseProxy
	l       sync.Mutex
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
