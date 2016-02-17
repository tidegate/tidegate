package core

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

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

type ReverseProxy interface {
	ServeHTTP(req *http.Request)
}

type MultipleEndpointReverseProxy struct {
	endpoints []*url.URL
}

func NewMutlipleEndpointReverseProxy() (res *MultipleEndpointReverseProxy) {
	res = &MultipleEndpointReverseProxy{}
	return
}

func (self *MultipleEndpointReverseProxy) ServeHTTP(req *http.Request) {
	if len(self.endpoints) > 0 {
		endpoint := self.endpoints[rand.Int()%len(self.endpoints)]
		req.URL.Scheme = endpoint.Scheme
		req.URL.Host = endpoint.Host
		req.URL.Path = singleJoiningSlash(endpoint.Path, req.URL.Path)
	}
}

func (self *MultipleEndpointReverseProxy) AddEndpoint(endpoint *url.URL) {
	self.endpoints = append(self.endpoints, endpoint)
}

type DomainSwitcher struct {
	proxies map[string]*MultipleEndpointReverseProxy
}

func NewDomainSwitcher() (res *DomainSwitcher) {
	res = &DomainSwitcher{}
	res.proxies = make(map[string]*MultipleEndpointReverseProxy)
	return
}

func (self *DomainSwitcher) AddNewDomain(domain string) (res *MultipleEndpointReverseProxy) {

	if proxy, ok := self.proxies[domain]; !ok {
		res = NewMutlipleEndpointReverseProxy()
		self.proxies[domain] = res
	} else {
		res = proxy
	}

	return
}

func (self *DomainSwitcher) GetDomain(domain string) (res *MultipleEndpointReverseProxy, err error) {
	if res, found := self.proxies[domain]; !found {
		err = errors.New(fmt.Sprintf("Unable to find reverse proxy for domain '%s'", domain))
	} else {
		res = res
	}
	return
}

func (self *DomainSwitcher) ServeHTTP(req *http.Request) {
	domain, _, _ := net.SplitHostPort(req.Host)
	if proxy, ok := self.proxies[domain]; ok {
		proxy.ServeHTTP(req)
	} else {
		fmt.Printf("nok")
	}
}

func NewMultipleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	// targetQuery := target.RawQuery
	domainSwitcher := NewDomainSwitcher()
	rp := domainSwitcher.AddNewDomain("blog.popular-design.fr")
	rp.AddEndpoint(&url.URL{
		Scheme: "http",
		Host:   "localhost:8085",
	})
	rp = domainSwitcher.AddNewDomain("acebedo.fr")
rp.AddEndpoint(&url.URL{
		Scheme: "http",
		Host:   "localhost:8085",
	})
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			domainSwitcher.ServeHTTP(req)
		}}
}
