// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package proxypool offers a proxy ranking model.
package proxypool

import (
	"log"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lifenjoiner/ewma"
	"github.com/lifenjoiner/pd/checker"
)

const ewmaSlide int = 10
const updateInterval time.Duration = 3 * time.Minute

// Proxy stores the upstream proxy(socks/http, h3) settings.
type Proxy struct {
	Ewma *ewma.EWMA
	URL  *url.URL
	url  string
}

// Dup duplicates a Proxy struct.
func (p *Proxy) Dup() *Proxy {
	e := *p.Ewma
	u := *p.URL
	s := p.url
	return &Proxy{&e, &u, s}
}

// Check the avalability of a Proxy.
func (p *Proxy) Check(target *url.URL, timeout time.Duration) error {
	ck := checker.NewTargetChecker(p.URL, timeout, nil, target)
	return ck.Check()
}

// NewProxy generates a Proxy form a URL string.
func NewProxy(s string) (*Proxy, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	return &Proxy{
		Ewma: ewma.NewMovingAverage(ewmaSlide),
		URL:  u,
		url:  s,
	}, nil
}

// NewProxies generates Proxies form URL strings.
func NewProxies(urls []string) []*Proxy {
	var proxies []*Proxy
	for _, s := range urls {
		if len(s) == 0 {
			continue
		}
		if !strings.Contains(s, "//") {
			s = "//" + s
		}
		p, err := NewProxy(s)
		if err == nil {
			proxies = append(proxies, p)
		}
	}
	return proxies
}

// ProxyPool struct.
type ProxyPool struct {
	sync.RWMutex
	Proxies       []*Proxy
	Checker       string
	ProxyProbeURL *url.URL
	Timeout       time.Duration
}

// GetProxy gets a proxy by index mapping.
func (pp *ProxyPool) GetProxy(i int) (p *Proxy) {
	pp.RLock()
	n := len(pp.Proxies)
	if n > 0 {
		p = pp.Proxies[i%n]
	}
	pp.RUnlock()
	return
}

// Update and sort the EWMA of proxies in the pool.
func (pp *ProxyPool) Update() {
	N := len(pp.Proxies)
	if N <= 0 {
		return
	}

	pp.RLock()
	proxies := make([]*Proxy, N)
	copy(proxies, pp.Proxies) // Both are slice!
	pp.RUnlock()

	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		p := proxies[i]
		go func() {
			defer wg.Done()
			startTime := time.Now()
			// Dial -> Handshake -> Transfer
			d := 3 * pp.Timeout
			if p.Check(pp.ProxyProbeURL, pp.Timeout) == nil {
				d = time.Since(startTime)
			}
			p.Ewma.Add(float64(d))
		}()
	}
	wg.Wait()

	pp.Lock()
	sort.Slice(pp.Proxies, func(i, j int) bool {
		return pp.Proxies[i].Ewma.Value() < pp.Proxies[j].Ewma.Value()
	})
	log.Printf("[ProxyPool] Sorted latencies:")
	for _, p := range pp.Proxies {
		log.Printf("[ProxyPool]  %v %s://%s", time.Duration(p.Ewma.Value()), p.URL.Scheme, p.URL.Host)
	}
	pp.Unlock()
}

// InitProxyPool initializes a ProxyPool from configured URLs.
func InitProxyPool(urls string, test string, timeout time.Duration) (pp map[string]*ProxyPool) {
	pp = make(map[string]*ProxyPool)

	ut, err := url.Parse(test)
	if err != nil {
		log.Printf("[ProxyPool] %v", err)
		return
	}

	proxies := NewProxies(strings.Split(urls, ","))
	if len(proxies) <= 0 {
		log.Printf("[ProxyPool] no proxy")
		return
	}

	allowedSchemes := [3]string{"http", "socks5", "socks4a"}
	for _, p := range proxies {
		p.URL.Scheme = strings.ToLower(p.URL.Scheme)
		s := p.URL.Scheme
		switch s {
		case "http", "socks5", "socks4a":
			if pp[s] == nil {
				pp[s] = &ProxyPool{}
			}
			pp[s].Lock()
			pp[s].Proxies = append(pp[s].Proxies, p)
			pp[s].Unlock()
		case "":
			for _, ss := range allowedSchemes {
				pe := p.Dup()
				pe.URL.Scheme = ss
				if pp[ss] == nil {
					pp[ss] = &ProxyPool{}
				}
				pp[ss].Lock()
				pp[ss].Proxies = append(pp[ss].Proxies, pe)
				pp[ss].Unlock()
			}
		default:
			log.Printf("[ProxyPool] unsupported proxy: %v", p.url)
		}
	}
	for i := 0; i < len(allowedSchemes); i++ {
		s := allowedSchemes[i]
		if pp[s] == nil {
			continue
		}
		pp[s].Lock()
		pp[s].ProxyProbeURL = ut
		pp[s].Timeout = timeout
		pp[s].Unlock()
		go func() {
			for {
				log.Printf("[ProxyPool] %v updating ...", s)
				pp[s].Update()
				time.Sleep(updateInterval)
			}
		}()
	}
	return
}
