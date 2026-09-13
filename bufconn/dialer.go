// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package bufconn offers access to connections with buffer.
package bufconn

import (
	"errors"
	// "log"
	"net"
	"net/netip"
	"net/url"
	"time"
)

// Bind config.
type BindConf struct {
	Interface *net.Interface
	IP        net.IP
}

// Direct dialer.
type Direct struct {
	BindConf
	nd net.Dialer
}

// BindConf is the bind interface config for dialer.
var BindCfg *BindConf

// NewDirect returns a Direct dialer.
func NewBindConf(filter string) (b *BindConf, err error) {
	b = &BindConf{}
	if filter != "" {
		if ip := net.ParseIP(filter); ip != nil {
			b.IP = ip
		} else {
			var ifi *net.Interface
			ifi, err = net.InterfaceByName(filter)
			if err == nil {
				b.Interface = ifi
			} else {
				err = errors.New(err.Error() + ": " + filter)
			}
		}
	}
	return
}

// DialTimeout dials with timeout. net.Dialer doesn't have this method.
func (d *Direct) DialTimeout(network, address string, timeout time.Duration) (c net.Conn, err error) {
	d.nd.Timeout = timeout
	return d.Dial(network, address)
}

// DialTimeout dials with timeout. net.Dialer doesn't have this method.
func (d *Direct) Dial(network, address string) (c net.Conn, err error) {
	var ipVer byte
	ok := d.Interface == nil
	if !ok {
		ipVer := ipVersion(getAddressIP(address))
		if d.IP != nil {
			if ipVer == 0 || ipVer == ipVersion(d.IP) {
				ok = true
			} else {
				err = errors.New("Dial: IP version mismatch")
			}
		}
	}
	if ok {
		// log.Printf("ip: %s", d.IP)
		c, err = d.dial(network, address, d.IP)
		if err == nil {
			return
		}
	}
	if d.Interface == nil {
		return
	}

	ips := d.InterfaceOuterIPs()
	if len(ips) == 0 {
		err = errors.New("Dial: can't get IPs of interface: " + d.Interface.Name)
		return
	}
	for _, ip := range ips {
		if ipVer != 0 && (ipVer != ipVersion(ip) || ip.Equal(d.IP)) {
			continue
		}
		// log.Printf("name: %v", ip)
		c, err = d.dial(network, address, ip)
		if err == nil {
			d.IP = ip // reuse?
			return
		}
	}

	return
}

func (d *Direct) dial(network, address string, ip net.IP) (net.Conn, error) {
	if ip != nil {
		switch network {
		case "tcp":
			d.nd.LocalAddr = &net.TCPAddr{IP: ip}
		case "udp":
			d.nd.LocalAddr = &net.UDPAddr{IP: ip}
		}
	}

	if d.Interface != nil {
		d.nd.Control = bindToDevice(d.Interface)
	}

	return d.nd.Dial(network, address)
}

// InterfaceOuterIPs returns ip addresses of the specified interface.
func (d *Direct) InterfaceOuterIPs() (ips []net.IP) {
	ipNets, err := d.Interface.Addrs()
	if err != nil {
		return
	}
	// log.Printf("%v", ipNets)
	var (
		ipv6 net.IP
		ipv4 net.IP
	)
	for _, ipNet := range ipNets {
		ip := ipNet.(*net.IPNet).IP
		if ip.IsLinkLocalUnicast() { // not link-local IPv6
			continue
		}
		// Prefer temporary IPv6 for privacy. Is Global Unicast got earlier?
		// Sometimes, IPv4 is IPv4-mapped IPv6!
		if ip.To4() != nil {
			ipv4 = ip
		} else {
			ipv6 = ip
		}
	}
	if ipv6 != nil {
		ips = append(ips, ipv6)
	}
	if ipv4 != nil {
		ips = append(ips, ipv4)
	}
	// log.Printf("%v", ips)
	return
}

func getAddressIP(address string) net.IP {
	h, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil
	}
	return net.ParseIP(h)
}

func isLoopback(address string) bool {
	ip := getAddressIP(address)
	return ip != nil && ip.IsLoopback()
}

func ipVersion(ip net.IP) byte {
	ip2, ok := netip.AddrFromSlice(ip)
	if !ok {
		return 0
	} else if ip2.Is4() {
		return '4'
	} else {
		return '6'
	}
}

// DialTimeout dials the address with timeout.
func DialTimeout(network, address string, timeout time.Duration) (*Conn, error) {
	var (
		c   net.Conn
		err error
	)
	if BindCfg == nil || isLoopback(address) {
		c, err = net.DialTimeout(network, address, timeout)
	} else {
		d := &Direct{}
		d.Interface = BindCfg.Interface
		d.IP = BindCfg.IP
		c, err = d.DialTimeout(network, address, timeout)
	}
	var conn *Conn
	if err == nil {
		_ = c.SetDeadline(time.Now().Add(timeout))
		conn = NewConn(c)
	}
	return conn, err
}

// OS do Parallel Queries to DNS of all interfaces!

// DialURL dials the URL with timeout.
func DialURL(u *url.URL, d time.Duration) (*Conn, error) {
	h := u.Host
	if len(u.Port()) == 0 {
		h += ":" + u.Scheme
	}
	n := "tcp"
	if u.Scheme == "h3" {
		n = "udp"
	}
	return DialTimeout(n, h, d)
}

// DialProxyTimeout dials the URL with timeout.
func DialProxyTimeout(u *url.URL, d time.Duration) (cs ConnSolver, err error) {
	switch u.Scheme {
	case "http", "https":
		cs, err = DialHTTP(u, d)
	case "socks5":
		cs, err = DialSocks5(u, d)
	case "socks4a":
		cs, err = DialSocks4a(u, d)
	default:
		err = errors.New("Unknown proxy scheme: " + u.Scheme)
	}
	return
}
