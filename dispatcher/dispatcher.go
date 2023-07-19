// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

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

// What a dispatcher instance is composed with.
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
	maxProxyTry int
	proxyTried  int
}

// The Dispatcher constructor.
func New(s string, c *bufconn.Conn, h string, p string, d time.Duration) *Dispatcher {
	return &Dispatcher{
		ServerType: s,
		Client:     c,
		DestHost:   h,
		DestPort:   p,
		Timeout:    d,
	}
}

// The main dispatcher, that dispatches how a client connection will be served.
func (d *Dispatcher) Dispatch(req protocol.Requester) bool {
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

	logPre := "[" + d.ServerType + "] " + req.Command() + " " + req.Target() + " <- " + d.Client.RemoteAddr().String()
	log.Printf("%v [type:%v]", logPre, strategy)

	var restart bool
	var err error
	ok := false
	v := 0.0
	h := d.DestHost + ":" + d.DestPort
	for d.tried = 0; d.tried < d.maxTry; d.tried++ {
		restart, err = d.ServeDirect(req)
		if err == nil {
			ok = true
			v = 1.0
			break
		} else if restart {
			// failed after the 2nd client packet has been sent following ServerHello
			break
		}
		// dialing or receiving ServerHello failed
	}
	if globalOnline && d.maxTry > 0 && strategy == statichost.StaticNil {
		GlobalHostStats.Update(h, v)
	}
	if ok || restart {
		return ok
	}

	for d.proxyTried = 0; d.proxyTried < d.maxProxyTry; d.proxyTried++ {
		restart, err = d.ServeProxied(req)
		if err == nil {
			return true
		} else if restart {
			return false
		}
	}

	ok = false
	if d.maxTry == 0 {
		log.Printf("%v <= no proxy succeeded, try direct", logPre)
		d.maxTry = 1
		v = 0.0
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

// Decide whether the host is aways go direct or proxied.
func (d *Dispatcher) DispatchByStaticRules() statichost.Strategy {
	return GlobalStaticHosts.GetStrategy(d.DestHost)
}

// Solve the direct connect tries by HostStat.
func (d *Dispatcher) DispatchByStats() {
	h := d.DestHost + ":" + d.DestPort
	stat := GlobalHostStats.GetStat(h)
	if stat.Count == 0 || stat.Value > 0.8 {
		d.maxTry = 3
	} else if stat.Value > 0.6 {
		d.maxTry = 2
	} else if stat.Value > 0.4 || stat.Count <= hoststat.EwmaSlide {
		d.maxTry = 1
	} else if stat.Value > 0.3 && time.Since(stat.Time) > 5*time.Minute {
		d.maxTry = 1
	} else if stat.Value > 0.2 && time.Since(stat.Time) > 7*time.Minute {
		d.maxTry = 1
	} else if stat.Value > 0.1 && time.Since(stat.Time) > 13*time.Minute {
		d.maxTry = 1
	} else if time.Since(stat.Time) > 31*time.Minute {
		d.maxTry = 1
	} else {
		d.maxTry = 0
	}
}

// The helper struct for DispatchIP.
type goodConn struct {
	sync.RWMutex
	c   net.Conn
	err error
	n   int
}

// Get the quickest responsive IP for a direct connection.
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
	for _, IP := range IPs {
		go func(ip string) {
			c, err := net.DialTimeout("tcp", net.JoinHostPort(ip, d.DestPort), d.Timeout)
			goodConn.Lock()
			if goodConn.c == nil {
				if err == nil { // bad ip returns fast too
					goodConn.c = c
				} else if goodConn.n == 1 { // return the last err (timeout)
					goodConn.err = err
				}
			} else if err == nil {
				// ESTABLISHED
				_ = c.SetDeadline(time.Now())
				c.Close()
			}
			goodConn.n--
			goodConn.Unlock()
		}(IP)
	}

	var conn *bufconn.Conn
	for {
		goodConn.RLock()
		if goodConn.c != nil {
			conn = bufconn.NewConn(goodConn.c)
		} else if goodConn.n <= 0 {
			err = goodConn.err
		}
		goodConn.RUnlock()
		if conn != nil || err != nil {
			break
		}
		time.Sleep(time.Nanosecond)
	}

	return conn, err
}

// Get the best proxy Conn.
func (d *Dispatcher) DispatchProxy() (cs bufconn.ConnSolver, proxy proxypool.Proxy, err error) {
	ProxyPool := GlobalProxyPool[d.ServerType]
	if ProxyPool == nil {
		err = errors.New("no valid proxy")
		return
	}
	proxy = ProxyPool.GetProxy(d.proxyTried)
	if proxy.URL != nil {
		switch d.ServerType {
		case "http":
			cs, err = bufconn.DialHTTP(proxy.URL, ProxyPool.Timeout)
		case "socks5":
			cs, err = bufconn.DialSocks5(proxy.URL, ProxyPool.Timeout)
		case "socks4a":
			cs, err = bufconn.DialSocks4a(proxy.URL, ProxyPool.Timeout)
		}
		if proxy.URL.User == nil {
			return
		} else {
			c := cs.GetConn()
			_ = c.SetDeadline(time.Now())
			c.Close()
			err = errors.New("proxy authentication is not implemented")
		}
	} else {
		err = errors.New("no valid proxy")
	}
	return
}

// Serve the client by direct connecting to the server.
func (d *Dispatcher) ServeDirect(req protocol.Requester) (bool, error) {
	client := d.Client
	logPre := fmt.Sprintf("[%v] direct:%v/%v %v %v", d.ServerType, d.tried+1, d.maxTry, req.Command(), req.Target())
	_ = client.SetDeadline(time.Now().Add(2 * d.Timeout))
	if req.Command() == "CONNECT" {
		err := req.GetRequest(client, client.R)
		if err != nil {
			log.Printf("%v <- %v <= TLS: no ClientHello, drop it.", logPre, client.RemoteAddr())
			return true, err
		}
	}
	restart := false
	c, err := d.DispatchIP()
	if err == nil {
		log.Printf("%v => %v <-> %v <-> %v", logPre, client.RemoteAddr(), c.LocalAddr(), c.RemoteAddr())
		fw := &forwarder.Forwarder{
			LeftAddr:  client.RemoteAddr(),
			LeftConn:  client,
			RightAddr: c.RemoteAddr(),
			RightConn: c,
			Timeout:   d.Timeout,
		}
		restart, err = req.Request(fw, d.tried == 1)
		c.Close()
	} else if IsDnsErr(err) {
		// Trust the specified DNS.
		// If the DNS isn't reliable enough, place a host in `blocked` to go proxied directly.
		// Host mapping `0.0.0.0` or `::` error: The requested name is valid, but no data of the requested type was found.
		// Disabled domain error: no such host
		// Host mapping `127.0.0.1` or `::1` is valid!
		_, err = client.Write([]byte("HTTP/1.1 569 DNS Orz\r\n\r\n"))
	}
	if err != nil {
		log.Printf("%v <= %v", logPre, err)
	}
	return restart, err
}

// Serve the client by proxy.
func (d *Dispatcher) ServeProxied(req protocol.Requester) (bool, error) {
	client := d.Client
	logPre := fmt.Sprintf("[%v] proxy:%v/%v %v %v", d.ServerType, d.proxyTried+1, d.maxProxyTry, req.Command(), req.Target())
	_ = client.SetDeadline(time.Now().Add(2 * d.Timeout))
	if req.Command() == "CONNECT" {
		err := req.GetRequest(client, client.R)
		if err != nil {
			log.Printf("%v <- %v <= TLS: no ClientHello, drop it.", logPre, client.RemoteAddr())
			return true, err
		}
	}
	restart := false
	conn, p, err := d.DispatchProxy()
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
			}
			restart, err = req.Request(fw, false)
		}
		c.Close()
	}
	if err != nil {
		log.Printf("%v <= %v", logPre, err)
	}
	return restart, err
}

func NotInternetHost(h string) bool {
	if statichost.HostIsIP(h) {
		IP := net.ParseIP(h)
		return !IP.IsGlobalUnicast() || IP.IsPrivate()
	} else {
		return !strings.Contains(h, ".")
	}
}

// Check if we are online.
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
	} else {
		log.Print(err)
	}
	log.Printf("[dispatcher] No probing URL available, always act as online!")
}
