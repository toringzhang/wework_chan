package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func ReverseProxyHandler(targetHost string) func(http.ResponseWriter, *http.Request) {
	uri, _ := url.Parse(targetHost)
	proxy := httputil.NewSingleHostReverseProxy(uri)
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Host = uri.Host
		r.URL.Scheme = "https"
		r.Host = uri.Host
		proxy.ServeHTTP(w, r)
	}
}
