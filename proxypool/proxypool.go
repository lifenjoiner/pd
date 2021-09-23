// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// A proxy ranking model.
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
	Url  string
}

func (p *Proxy) Dup() *Proxy {
	e := *p.Ewma
	u := *p.URL
	s := p.Url
	return &Proxy{&e, &u, s}
}

func (p *Proxy) Dial(d time.Duration) error {
	return nil
}

func (p *Proxy) Check(target *url.URL, d time.Duration) error {
	ck := &checker.TargetChecker{p.URL, d, nil, target}
	return ck.Check()
}

func NewProxy(s string) (*Proxy, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	return &Proxy{
		Ewma: ewma.NewMovingAverage(ewmaSlide),
		URL:  u,
		Url:  s,
	}, nil
}

func NewProxies(urls []string) []*Proxy {
	var proxies []*Proxy
	for i := 0; i < len(urls); i++ {
		s := urls[i]
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

type ProxyPool struct {
	sync.RWMutex
	Proxies       []*Proxy
	Checker       string
	ProxyProbeUrl *url.URL
	Timeout       time.Duration
}

// Update and sort the EWMA of proxies in the pool.
func (pp *ProxyPool) Update() {
	p := pp.Proxies
	N := len(p)
	if N <= 0 {
		return
	}
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		proxy := p[i]
		go func() {
			defer wg.Done()
			startTime := time.Now()
			// Dial -> Handshake -> Transfer
			d := 3 * pp.Timeout
			if proxy.Check(pp.ProxyProbeUrl, pp.Timeout) == nil {
				d = time.Since(startTime)
			}
			proxy.Ewma.Add(float64(d))
		}()
	}
	wg.Wait()

	pp.Lock()
	sort.Slice(p, func(i, j int) bool {
		return p[i].Ewma.Value() < p[j].Ewma.Value()
	})
	log.Printf("[ProxyPool] Sorted latencies:")
	for i := 0; i < N; i++ {
		log.Printf("[ProxyPool]  %v %s://%s", time.Duration(p[i].Ewma.Value()), p[i].URL.Scheme, p[i].URL.Host)
	}
	pp.Unlock()
}

// Initialize a ProxyPool from configured URLs.
func InitProxyPool(urls string, test string, d time.Duration) (pp map[string]*ProxyPool) {
	pp = make(map[string]*ProxyPool)
	Url := strings.Split(urls, ",")
	ut, err := url.ParseRequestURI(test)
	if err != nil {
		log.Printf("[ProxyPool] %v", err)
		return
	}

	proxies := NewProxies(Url)
	N := len(proxies)
	if N == 0 {
		log.Printf("[ProxyPool] no proxy")
		return
	}

	allowedSchemes := [3]string{"http", "socks5", "socks4a"}
	for i := 0; i < N; i++ {
		proxies[i].URL.Scheme = strings.ToLower(proxies[i].URL.Scheme)
		s := proxies[i].URL.Scheme
		switch s {
		case "http", "socks5", "socks4a":
			if pp[s] == nil {
				pp[s] = &ProxyPool{}
			}
			pp[s].Proxies = append(pp[s].Proxies, proxies[i])
		case "":
			for j := 0; j < len(allowedSchemes); j++ {
				scheme := allowedSchemes[j]
				p := proxies[i]
				ph := p.Dup()
				ph.URL.Scheme = scheme
				if pp[scheme] == nil {
					pp[scheme] = &ProxyPool{}
				}
				pp[scheme].Proxies = append(pp[scheme].Proxies, ph)
			}
		default:
			log.Printf("[ProxyPool] unsupported proxy: %v", proxies[i].Url)
		}
	}
	for j := 0; j < len(allowedSchemes); j++ {
		scheme := allowedSchemes[j]
		if pp[scheme] == nil {
			continue
		}
		pp[scheme].ProxyProbeUrl = ut
		pp[scheme].Timeout = d
		go func() {
			for {
				log.Printf("[ProxyPool] %v updating ...", scheme)
				pp[scheme].Update()
				time.Sleep(updateInterval)
			}
		}()
	}
	return
}
