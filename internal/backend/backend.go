package backend

import (
	"bytes"
	"fmt"
	"go-balancer/internal/balancer/config"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type backend struct {
	host string
	port int

	url *url.URL

	alive  bool
	rwLock sync.RWMutex
}

type BackendRef = *backend

// Creates a new backend from a BackendInfo object
func newBackend(info config.BackendInfo) *backend {
	return &backend{
		host:  info.Host,
		port:  info.Port,
		url:   info.URL,
		alive: true,
	}
}

// Copies from another backend with a new mutex
func copyBackend(other *backend) backend {
	return backend{
		host:  other.host,
		port:  other.port,
		url:   other.url,
		alive: true,
	}
}

func (b *backend) Equal(other *backend) bool {
	return b.host == other.host && b.port == other.port
}

func (b *backend) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("{\"host\":\"%s\",\"port\":\"%d\",\"alive\":%t}", b.host, b.port, b.alive)), nil
}

func (b *backend) GetURL() *url.URL {
	return b.url
}

func (b *backend) GetAlive() bool {
	b.rwLock.RLock()
	defer b.rwLock.RUnlock()

	return b.alive
}

func (b *backend) setAlive(alive bool) {
	b.rwLock.Lock()

	b.alive = alive

	b.rwLock.Unlock()
}

type reverseProxyErrorHandler = func(http.ResponseWriter, *http.Request, error)

func (b *backend) newProxy(errorHandler reverseProxyErrorHandler) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(b.url)
	proxy.ErrorHandler = errorHandler

	return proxy
}

func (b *backend) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	var proxyError error = nil

	// Create a new proxy for the backend, attaching an error handler
	proxy := b.newProxy(func(w http.ResponseWriter, r *http.Request, err error) {
		proxyError = err
	})

	// Use the proxy to serve the request
	proxy.ServeHTTP(w, r)

	return proxyError
}

type ReadonlyBackendList struct {
	list *[]*backend
}

func (l ReadonlyBackendList) Len() int {
	return len(*l.list)
}

func (l ReadonlyBackendList) Get(index int) *backend {
	return (*l.list)[index]
}

// Linear search for the element
// Returns -1 if not present
func (l ReadonlyBackendList) IndexOf(other *backend) int {
	for i := range *l.list {
		if (*l.list)[i].Equal(other) {
			return i
		}
	}
	return -1
}

func (l ReadonlyBackendList) MarshalJSON() ([]byte, error) {
	encodedObjs := make([][]byte, len(*l.list))

	for i := range *l.list {
		encodedObjs[i], _ = (*l.list)[i].MarshalJSON()
	}

	encodedList := bytes.Join(encodedObjs, []byte(","))

	final := make([]byte, len(encodedList)+2)
	final[0] = '['
	final[len(final)-1] = ']'
	copy(final[1:], encodedList)

	return final, nil
}
