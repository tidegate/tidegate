package core

import (
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"
)

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

/*******************************************************************************/
// MultipleEndpointReverseProxy
/*******************************************************************************/
type MultipleEndpointReverseProxy struct {
	endpoints map[string]*url.URL
}

func NewMutlipleEndpointReverseProxy() (res *MultipleEndpointReverseProxy) {
	res = &MultipleEndpointReverseProxy{}
	res.endpoints = make(map[string]*url.URL)
	return
}

func (self *MultipleEndpointReverseProxy) ServeHTTP(req *http.Request) {
	if len(self.endpoints) > 0 {
		var keys []string
		for k, _ := range self.endpoints {
			keys = append(keys, k)
		}
		endpoint := self.endpoints[keys[rand.Int()%len(keys)]]
		logger.Debugf("Request forwarded to endpoint: %v", endpoint)
		req.URL.Scheme = endpoint.Scheme
		req.URL.Host = endpoint.Host

		singleJoiningSlash := func(a, b string) string {
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

		req.URL.Path = singleJoiningSlash(endpoint.Path, req.URL.Path)
	}
}

func (self *MultipleEndpointReverseProxy) AddEndpoint(endpoint *url.URL) {
	logger.Debugf("Added endpoint: %v", endpoint)
	self.endpoints[endpoint.String()] = endpoint
}

func (self *MultipleEndpointReverseProxy) RemoveEndpoint(endpoint *url.URL) {
	logger.Debugf("Removed endpoint: %v", endpoint)
	delete(self.endpoints, endpoint.String())
}

/*******************************************************************************/
// ProxyMapEntry
/*******************************************************************************/
type ProxyMapEntry struct {
	re    *regexp.Regexp
	proxy *MultipleEndpointReverseProxy
}

type ProxyMap map[string][]ProxyMapEntry

type ProxyHandle struct {
	proxies ProxyMap
}

func NewProxyHandle() (res *ProxyHandle) {
	res = &ProxyHandle{}
	res.proxies = make(ProxyMap)
	return
}

func (self *ProxyHandle) ServeHTTP(req *http.Request) {
	domain := req.Host
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
	} else {
		logger.Errorf("No proxies found for domain %v", domain)
	}
}

func (self *ProxyHandle) AddReverseProxy(domain string, location *regexp.Regexp) (res *MultipleEndpointReverseProxy) {
	res = NewMutlipleEndpointReverseProxy()
	self.proxies[domain] = append(self.proxies[domain], ProxyMapEntry{location, res})
	logger.Debugf("Added reverse proxy for domain: %v", domain)
	return
}

func (self *ProxyHandle) RemoveReverseProxy(domain string) {
	delete(self.proxies, domain)
}

func (self *ProxyHandle) Contains(domain string) (res bool) {
	_, res = self.proxies[domain]
	return
}

func (self *ProxyHandle) GetReverseProxies(domain string) (res *[]ProxyMapEntry, err error) {
	if proxies, found := self.proxies[domain]; !found {
		err = errors.New(fmt.Sprintf("Unable to find reverse proxy for domain '%s'", domain))
	} else {
		res = &proxies
	}
	return
}

/*******************************************************************************/
// TidegateServerEngine
/*******************************************************************************/
type TidegateServerEngine interface {
	GetListener() net.Listener
}

type TidegateServerEngineFactory interface {
	Instantiate(ln net.Listener) TidegateServerEngine
}

/*******************************************************************************/
// TidegateServer
/*******************************************************************************/
type TidegateServer struct {
	server      *http.Server
	ProxyHandle *ProxyHandle
	engine      TidegateServerEngine
}

func NewTidegateServer(hostPort string) (res *TidegateServer) {
	res = &TidegateServer{}
	res.ProxyHandle = NewProxyHandle()
	logger.Debugf("New tidegateserver  !, %v", hostPort)
	res.server = &http.Server{
		Addr: hostPort,
		Handler: &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				res.ProxyHandle.ServeHTTP(req)
			}}}
	return
}

func (self *TidegateServer) ListenAndServe(engineFactory TidegateServerEngineFactory) (err error) {
	logger.Debugf("Server Started!")
	ln, err := net.Listen("tcp", self.server.Addr)
	if err == nil {
		self.engine = engineFactory.Instantiate(ln)
		go self.server.Serve(self.engine.GetListener())
	}
	return
}

func (self *TidegateServer) AddEndpoint(inUrl *url.URL, outUrl *url.URL) (err error) {
	inHost, inPort, err := net.SplitHostPort(inUrl.Host)
	if err != nil {
		return
	}
	_, serverPort, err := net.SplitHostPort(self.server.Addr)
	if err != nil {
		return
	}
	if inPort != serverPort {
		err = errors.New(fmt.Sprintf("Cannot add endpoint '%v'->'%v': server port is '%v' cannot be mapped on '%v", inUrl, outUrl, serverPort, inPort))
	}
	v := reflect.ValueOf(self.engine)
	v2 := reflect.ValueOf((*TidegateTLSServerEngine)(nil))
	if inUrl.Scheme == "https" && (v.Kind() != v2.Kind()) {
		err = errors.New(fmt.Sprintf("Cannot add endpoint '%v'->'%v': endpoint declared as SSL but engine is SSL", inUrl, outUrl))
	}

	var re *regexp.Regexp
	re, err = regexp.Compile(outUrl.RawPath)
	if err != nil {
		return
	}
	proxy := self.ProxyHandle.AddReverseProxy(inHost, re)
	proxy.AddEndpoint(outUrl)
	return
}

/*******************************************************************************/
// TidegateSimpleServerEngine
/*******************************************************************************/
type TidegateSimpleServerEngine struct {
	listener *tcpKeepAliveListener
}

func NewTidegateSimpleServerEngine(ln net.Listener) (res *TidegateSimpleServerEngine) {
	res = &TidegateSimpleServerEngine{}
	res.listener = &tcpKeepAliveListener{ln.(*net.TCPListener)}
	return
}

func (self *TidegateSimpleServerEngine) GetListener() net.Listener {
	return self.listener
}

type TidegateSimpleServerEngineFactory struct{}

func (self *TidegateSimpleServerEngineFactory) Instantiate(ln net.Listener) (res TidegateServerEngine) {
	res = NewTidegateSimpleServerEngine(ln)
	return
}

/*******************************************************************************/
// TidegateTLSServerEngine
/*******************************************************************************/
type TidegateTLSServerEngine struct {
	listener net.Listener
	config   tls.Config
}

func (self *TidegateTLSServerEngine) GetListener() net.Listener {
	return self.listener
}

func NewTidegateTLSServerEngine(ln net.Listener) (res *TidegateTLSServerEngine) {

	res = &TidegateTLSServerEngine{}
	res.config.NextProtos = []string{"http/1.1"}
	res.listener = tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, &res.config)

	return
}

func (self *TidegateTLSServerEngine) AddCertificate(cert_path string, key_path string, err error) {
	cert, err := tls.LoadX509KeyPair(cert_path, key_path)
	if err == nil {
		self.config.Certificates = append(self.config.Certificates, cert)
		self.config.BuildNameToCertificate()
	}
	return
}

type TidegateTLSServerEngineFactory struct{}

func (self *TidegateTLSServerEngineFactory) Instantiate(ln net.Listener) (res TidegateServerEngine) {
	res = NewTidegateTLSServerEngine(ln)
	return
}
