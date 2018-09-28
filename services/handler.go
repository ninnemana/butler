package services

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type handler struct{}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	if fn, ok := hostProxy[host]; ok {
		fn.ServeHTTP(w, r)
		return
	}

	if target, ok := hostTarget[host]; ok {
		remote, err := url.Parse(target)
		if err != nil {
			log.Println("target parse fail:", err)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		hostProxy[host] = proxy
		proxy.ServeHTTP(w, r)
		return
	}
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
