// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package dispatcher decides if a client is served directly or by a proxy.
package dispatcher

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/checker"
	"github.com/lifenjoiner/pd/forwarder"
	"github.com/lifenjoiner/pd/hoststat"
	"github.com/lifenjoiner/pd/protocol"
	"github.com/lifenjoiner/pd/protocol/http"
	"github.com/lifenjoiner/pd/proxypool"
	"github.com/lifenjoiner/pd/statichost"
)

// The global parameters for dispatcher.
var (
	GlobalStaticHosts statichost.StaticHosts
	GlobalHostStats   *hoststat.HostStats
	GlobalProxyPool   map[string]*proxypool.ProxyPool
)

// If we are offline, don't update the GlobalHostStats.
var globalOnline bool

// Dispatcher struct is what a dispatcher instance is composed with.
type Dispatcher struct {
	ServerType   string
	Client       *bufconn.Conn
	DestHost     string
	DestPort     string
	Timeout      time.Duration
	ParallelDial bool
	//local
	maxTry      int
	tried       int
	directWave  float64
	maxProxyTry int
	proxyTried  int
}

// New generates a new Dispatcher.
func New(s string, c *bufconn.Conn, h string, p string, d time.Duration) *Dispatcher {
	return &Dispatcher{
		ServerType: s,
		Client:     c,
		DestHost:   h,
		DestPort:   p,
		Timeout:    d,
	}
}

// Dispatch is the main dispatcher, that dispatches how a client connection will be served.
func (d *Dispatcher) Dispatch(req protocol.Requester) bool {
	d.directWave = 1

	var strategy statichost.Strategy
	if NotInternetHost(d.DestHost) {
		log.Printf("[dispatcher] %v isn't Internet host, won't go proxied.", d.DestHost)
		strategy = statichost.StaticDirect
		d.maxTry = 3
		d.maxProxyTry = 0
	} else {
		d.maxProxyTry = 3
		strategy = d.DispatchByStaticRules()
		switch strategy {
		case statichost.StaticDirect:
			d.maxTry = 3
			d.maxProxyTry = 0
		case statichost.StaticBlocked:
			d.maxTry = 0
		default:
			d.DispatchByStats()
		}
	}

	logPre := "[" + d.ServerType + "] " + req.Command() + " " + req.Host() + " <- " + d.Client.RemoteAddr().String()
	log.Printf("%v [type:%v]", logPre, strategy)

	var restart bool // failed after the 2nd client packet has been sent following ServerHello
	var err error
	ok := false
	v := 0.0
	h := d.DestHost + ":" + d.DestPort
	for d.tried = 0; d.tried < d.maxTry; d.tried++ {
		restart, err = d.ServeDirect(req)
		if err == nil {
			ok = true
			v = 1.0
		}
		if globalOnline && strategy == statichost.StaticNil {
			GlobalHostStats.Update(h, v)
			if restart {
				GlobalHostStats.Update(h, v)
			}
		}
		if ok || restart {
			return ok
		}
		// dialing or receiving ServerHello failed
	}

	for d.proxyTried = 0; d.proxyTried < d.maxProxyTry; d.proxyTried++ {
		restart, err = d.ServeProxied(req)
		if err == nil {
			return true
		} else if restart {
			return false
		}
	}

	if d.maxTry == 0 {
		log.Printf("%v <= no proxy succeeded, try direct once", logPre)
		d.maxTry = 1
		_, err = d.ServeDirect(req)
		if err == nil {
			v = 1.0
			ok = true
		}
		if globalOnline && strategy == statichost.StaticNil {
			GlobalHostStats.Update(h, v)
		}
	}
	return ok
}

// DispatchByStaticRules decides whether the host is aways go direct or proxied.
func (d *Dispatcher) DispatchByStaticRules() statichost.Strategy {
	return GlobalStaticHosts.GetStrategy(d.DestHost)
}

// DispatchByStats solves the direct connecting tries by HostStat.
func (d *Dispatcher) DispatchByStats() {
	h := d.DestHost + ":" + d.DestPort
	stat := GlobalHostStats.GetStat(h)
	if stat.Count == 0 {
		stat.Value = 1
	}

	v := stat.Value
	d.directWave = v

	if v > 0.8 {
		d.maxTry = 3
	} else if v > 0.6 {
		d.maxTry = 2
	} else if v > 0.4 || stat.Count <= hoststat.EwmaSlide {
		d.maxTry = 1
	} else {
		dt := time.Since(stat.Time)
		if v > 0.3 && dt > 5*time.Minute {
			d.maxTry = 1
		} else if v > 0.2 && dt > 7*time.Minute {
			d.maxTry = 1
		} else if v > 0.1 && dt > 13*time.Minute {
			d.maxTry = 1
		} else if dt > 31*time.Minute {
			d.maxTry = 1
		} else {
			d.maxTry = 0
		}
	}
}

// goodConn is the helper struct for DispatchIP.
type goodConn struct {
	sync.RWMutex
	c   net.Conn
	err error
	n   int
}

// DispatchIP gets the quickest responded IP for a direct connection.
func (d *Dispatcher) DispatchIP() (*bufconn.Conn, error) {
	if !d.ParallelDial || (d.tried < 1 && d.maxTry > 1) || statichost.HostIsIP(d.DestHost) {
		c, err := net.DialTimeout("tcp", net.JoinHostPort(d.DestHost, d.DestPort), d.Timeout)
		if err != nil {
			return nil, err
		}
		return bufconn.NewConn(c), nil
	}

	// DNS/host filtering results host to "0.0.0.0" or "127.0.0.1".
	// For go, "0.0.0.0"/"::" are unspecified address that causes error. But "0.0.0.0" returns "0.0.0.0".
	// We trust reliable DNS lookup results (:
	IPs, err := net.LookupHost(d.DestHost)
	if err != nil {
		return nil, err
	}

	var goodConn goodConn
	goodConn.n = len(IPs)
	waitChannel := make(chan struct{})
	for i := 0; i < goodConn.n; i++ {
		ip := IPs[i]
		go func() {
			c, err := net.DialTimeout("tcp", net.JoinHostPort(ip, d.DestPort), d.Timeout)
			goodConn.Lock()
			if goodConn.c == nil {
				if err == nil { // bad ip returns fast too
					goodConn.c = c
					close(waitChannel)
				} else if goodConn.n == 1 { // return the last err (timeout)
					goodConn.err = err
					close(waitChannel)
				}
			} else if err == nil {
				// ESTABLISHED
				_ = c.SetDeadline(time.Now())
				c.Close()
			}
			goodConn.n--
			goodConn.Unlock()
		}()
	}

	<-waitChannel
	var conn *bufconn.Conn
	goodConn.RLock()
	if goodConn.c != nil {
		conn = bufconn.NewConn(goodConn.c)
	} else {
		err = goodConn.err
	}
	goodConn.RUnlock()

	return conn, err
}

// DispatchProxy gets the best proxy Conn.
func (d *Dispatcher) DispatchProxy() (cs bufconn.ConnSolver, pp *proxypool.ProxyPool, p *proxypool.Proxy, err error) {
	pp = GlobalProxyPool[d.ServerType]
	if pp == nil {
		err = errors.New("no valid proxy")
		return
	}
	p = pp.GetProxy(d.proxyTried)
	if p.URL != nil {
		switch d.ServerType {
		case "http":
			cs, err = bufconn.DialHTTP(p.URL, pp.Timeout)
		case "socks5":
			cs, err = bufconn.DialSocks5(p.URL, pp.Timeout)
		case "socks4a":
			cs, err = bufconn.DialSocks4a(p.URL, pp.Timeout)
		}
		if p.URL.User == nil {
			return
		}
		c := cs.GetConn()
		_ = c.SetDeadline(time.Now())
		c.Close()
		err = errors.New("proxy authentication is not implemented")
	} else {
		err = errors.New("no valid proxy")
	}
	return
}

// ServeDirect serves the client by direct connection to the server.
func (d *Dispatcher) ServeDirect(req protocol.Requester) (bool, error) {
	client := d.Client
	logPre := fmt.Sprintf("[%v] direct:%v/%v %v %v", d.ServerType, d.tried+1, d.maxTry, req.Command(), req.Host())
	_ = client.SetDeadline(time.Now().Add(2 * d.Timeout))
	var leftTran forwarder.Transformer
	if req.Command() == "CONNECT" {
		err := req.GetRequest(client, client.R)
		if err != nil {
			log.Printf("%v <- %v <= TLS: no ClientHello, drop it.", logPre, client.RemoteAddr())
			return true, err
		}
	} else {
		leftTran = &http.ReqestTransformer{}
	}
	restart := false
	c, err := d.DispatchIP()
	if err == nil {
		log.Printf("%v => %v <-> %v <-> %v", logPre, client.RemoteAddr(), c.LocalAddr(), c.RemoteAddr())
		wave := d.directWave
		if d.maxTry > 1 && d.tried < 1 {
			wave = 1.0
		}
		fw := &forwarder.Forwarder{
			LeftAddr:  client.RemoteAddr(),
			LeftConn:  client,
			LeftTran:  leftTran,
			RightAddr: c.RemoteAddr(),
			RightConn: c,
			Timeout:   d.Timeout,
			Wave:      wave,
		}
		restart, err = req.Request(fw, false, d.tried == d.maxTry>>1)
		c.Close()
	} else if IsDNSErr(err) {
		// Trust the specified DNS.
		// If the DNS isn't reliable enough, place a host in `blocked` to go proxied directly.
		// Host mapping `0.0.0.0` or `::` error: The requested name is valid, but no data of the requested type was found.
		// Disabled domain error: no such host
		// Host mapping `127.0.0.1` or `::1` is valid!
		if req.Command() == "CONNECT" {
			_, err = client.Write([]byte("\x15\x03\x03\x00\x02\x02\x00"))
		} else {
			_, err = client.Write([]byte("HTTP/1.1 569 DNS Orz\r\n\r\n"))
		}
	}
	if err != nil {
		log.Printf("%v <= %v", logPre, err)
	}
	return restart, err
}

// ServeProxied serves the client by proxy.
func (d *Dispatcher) ServeProxied(req protocol.Requester) (bool, error) {
	client := d.Client
	logPre := fmt.Sprintf("[%v] proxy:%v/%v %v %v", d.ServerType, d.proxyTried+1, d.maxProxyTry, req.Command(), req.Host())
	_ = client.SetDeadline(time.Now().Add(2 * d.Timeout))
	if req.Command() == "CONNECT" {
		err := req.GetRequest(client, client.R)
		if err != nil {
			log.Printf("%v <- %v <= TLS: no ClientHello, drop it.", logPre, client.RemoteAddr())
			return true, err
		}
	}
	restart := false
	conn, pp, p, err := d.DispatchProxy()
	if err == nil {
		c := conn.GetConn()
		log.Printf("%v => %v <-> %v <-> %v", logPre, client.RemoteAddr(), c.LocalAddr(), p.URL.Host)
		err = conn.Bond(req.Command(), req.Hostname(), req.Port(), nil)
		if err == nil {
			fw := &forwarder.Forwarder{
				LeftAddr:  client.RemoteAddr(),
				LeftConn:  client,
				RightAddr: c.RemoteAddr(),
				RightConn: c,
				Timeout:   d.Timeout,
				Wave:      1,
			}
			restart, err = req.Request(fw, true, false)
		}
		c.Close()
	}
	if err != nil {
		log.Printf("%v <= %v", logPre, err)
		if globalOnline && p != nil {
			pp.UpdateProxy(p, 3*pp.Timeout)
			pp.Sort()
		}
	}
	return restart, err
}

// NotInternetHost checks if the host is for public servers.
func NotInternetHost(h string) bool {
	if statichost.HostIsIP(h) {
		IP := net.ParseIP(h)
		return !IP.IsGlobalUnicast() || IP.IsPrivate()
	}
	return !strings.Contains(h, ".")
}

// StartProbeDirect checks if we are online.
func StartProbeDirect(url string, d time.Duration) {
	globalOnline = true
	ck, err := checker.New(url, d, "")
	if err == nil {
		go func() {
			for {
				globalOnline = ck.Check() == nil
				msg := "offline"
				if globalOnline {
					msg = "online"
				}
				log.Printf("[dispatcher] We are %v.", msg)
				time.Sleep(time.Minute)
			}
		}()
		return
	}
	log.Print(err)
	log.Printf("[dispatcher] No probing URL available, always act as online!")
}
