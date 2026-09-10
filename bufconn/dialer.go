// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package bufconn offers access to connections with buffer.
package bufconn

import (
	"errors"
	"net"
	"net/url"
	"time"
)

// DialTimeout dials the address with timeout.
func DialTimeout(network, address string, timeout time.Duration) (*Conn, error) {
	c, err := net.DialTimeout(network, address, timeout)
	var conn *Conn
	if err == nil {
		_ = c.SetDeadline(time.Now().Add(timeout))
		conn = NewConn(c)
	}
	return conn, err
}

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
