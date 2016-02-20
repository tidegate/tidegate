package core

import (
	"crypto/tls"
	//	"errors"
	//	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"
	"errors"
	"fmt"
)

func (self *MultipleEndpointReverseProxy) SingleJoiningSlash(a, b string) string {
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

type MultipleEndpointReverseProxy struct {
	endpoints map[string]*url.URL
}

func NewMutlipleEndpointReverseProxy() (res *MultipleEndpointReverseProxy) {
	res = &MultipleEndpointReverseProxy{}
	res.endpoints = make(map[string]*url.URL)
	return
}

func (self *MultipleEndpointReverseProxy) ServeHTTP(req *http.Request) {
	logger.Debugf("Request forwarding")
	if len(self.endpoints) > 0 {
		var keys []string
		for k, _ := range self.endpoints {
			keys = append(keys, k)
		}
		endpoint := self.endpoints[keys[rand.Int()%len(keys)]]
		logger.Debugf("Request forwarded to endpoint: %v", endpoint)
		req.URL.Scheme = endpoint.Scheme
		req.URL.Host = endpoint.Host
		req.URL.Path = self.SingleJoiningSlash(endpoint.Path, req.URL.Path)
	}
}

func (self *MultipleEndpointReverseProxy) AddEndpoint(endpoint *url.URL) {
	logger.Debugf("Added endpoint: %v", endpoint)
	self.endpoints[endpoint.String()] = endpoint
}

func (self *MultipleEndpointReverseProxy) RemoveEndpoint(endpoint *url.URL) {
	delete(self.endpoints, endpoint.String())
}

//type DomainSwitcher struct {
//	proxies map[string]*MultipleEndpointReverseProxy
//}
//
//func NewDomainSwitcher() (res *DomainSwitcher) {
//	res = &DomainSwitcher{}
//	res.proxies = make(map[string]*MultipleEndpointReverseProxy)
//	return
//}
//
//func (self *DomainSwitcher) AddNewDomain(domain string) (res *MultipleEndpointReverseProxy) {
//	if proxy, ok := self.proxies[domain]; !ok {
//		res = NewMutlipleEndpointReverseProxy()
//		self.proxies[domain] = res
//	} else {
//		res = proxy
//	}
//
//	return
//}
//
//func (self *DomainSwitcher) GetDomain(domain string) (res *MultipleEndpointReverseProxy, err error) {
//	if res, found := self.proxies[domain]; !found {
//		err = errors.New(fmt.Sprintf("Unable to find reverse proxy for domain '%s'", domain))
//	} else {
//		res = res
//	}
//	return
//}
//
//func (self *DomainSwitcher) ServeHTTP(req *http.Request) {
//	domain, _, _ := net.SplitHostPort(req.Host)
//	if proxy, ok := self.proxies[domain]; ok {
//		proxy.ServeHTTP(req)
//	} else {
//		fmt.Printf("nok")
//	}
//}

//func NewMultipleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
//	// targetQuery := target.RawQuery
//	domainSwitcher := NewDomainSwitcher()
//	rp := domainSwitcher.AddNewDomain("blog.popular-design.fr")
//	rp.AddEndpoint(&url.URL{
//		Scheme: "http",
//		Host:   "localhost:8085",
//	})
//	rp = domainSwitcher.AddNewDomain("acebedo.fr")
//rp.AddEndpoint(&url.URL{
//		Scheme: "http",
//		Host:   "localhost:8085",
//	})
//	return &httputil.ReverseProxy{
//		Director: func(req *http.Request) {
//			domainSwitcher.ServeHTTP(req)
//		}}
//}

type ProxyMapEntry struct {
	re    *regexp.Regexp
	proxy *MultipleEndpointReverseProxy
}

type ProxyMap map[string][]ProxyMapEntry

type TidegateServer struct {
	server  *http.Server
	proxies ProxyMap
}

func NewTidegateServer(hostPort string) (res *TidegateServer) {
	res = &TidegateServer{}
	logger.Debugf("New tidegateserver  !, %v", hostPort)
	res.proxies = make(ProxyMap)
	res.server = &http.Server{
		Addr: hostPort,
		Handler: &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				res.ServeHTTP(req)
			}}}
	return
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func (self *TidegateServer) ListenAndServe() (err error) {
	ln, err := net.Listen("tcp", self.server.Addr)
	if err == nil {
		logger.Debugf("Started!")
		go self.server.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
	} else {
		logger.Debugf("Not started %v", err)
	}
	return
}

func (self *TidegateServer) ServeHTTP(req *http.Request) {
	domain := req.Host
	logger.Debugf("Reverse !, %v", req.URL)
	if strings.Contains(req.Host, ":") {
		domain, _, _ = net.SplitHostPort(req.Host)
	}
	if proxies, ok := self.proxies[domain]; ok {
	  logger.Debugf("Found %v proxies for domain %v", len(proxies), domain)
PROXY_MATCHED:
		for idx, proxyEntry := range proxies {
			if proxyEntry.re.MatchString(req.URL.String()) {
			  logger.Debugf("Proxy %v is matching !", idx)
				proxyEntry.proxy.ServeHTTP(req)
				break PROXY_MATCHED
			}
		}
	} else  {
	  logger.Errorf("No proxies found for domain %v", domain)
	}
}

func (self *TidegateServer) AddReverseProxy(domain string, location *regexp.Regexp) (res *MultipleEndpointReverseProxy) {
	res = NewMutlipleEndpointReverseProxy()
	self.proxies[domain] = append(self.proxies[domain], ProxyMapEntry{location, res})
	logger.Debugf("Added reverse proxy for domain: %v", domain)
	return
}
func (self *TidegateServer) RemoveReverseProxy(domain string) {
	delete(self.proxies, domain)
}

func (self *TidegateServer) Contains(domain string) (res bool) {
	_, res = self.proxies[domain]
	return
}

func (self *TidegateServer) GetReverseProxies(domain string) (res *[]ProxyMapEntry, err error) {
	if proxies, found := self.proxies[domain]; !found {
		err = errors.New(fmt.Sprintf("Unable to find reverse proxy for domain '%s'", domain))
	} else {
		res = &proxies
	}
	return
}

type TidegateTLSServer struct {
	server  *http.Server
	config  *tls.Config
	proxies map[string]*MultipleEndpointReverseProxy
}

func NewTidegateTLSServer(hostPort string) (res *TidegateTLSServer, err error) {
	res = &TidegateTLSServer{}
	res.config = &tls.Config{}
	return
}

func (self *TidegateTLSServer) AddCertificate(cert_path string, key_path string, err error) {
	cert, err := tls.LoadX509KeyPair(cert_path, key_path)
	if err == nil {
		self.config.Certificates = append(self.config.Certificates, cert)
		self.config.BuildNameToCertificate()
	}
	return
}
