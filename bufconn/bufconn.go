// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package bufconn offers access to connections with buffer.
package bufconn

import (
	"bufio"
	"errors"
	"net"
	"net/url"
	"time"
)

// Conn is a deadline sensitive connection with bufio reader. Set the deadline before each IO.
type Conn struct {
	net.Conn
	R *bufio.Reader
}

// ReadData is non-blocking.
func (c *Conn) ReadData() ([]byte, error) {
	return ReadData(c.R)
}

// ReceiveData is blocking.
func (c *Conn) ReceiveData() ([]byte, error) {
	return ReceiveData(c.R)
}

// SplitWrite is to break length pattern.
func (c *Conn) SplitWrite(b []byte, x int) (n int, err error) {
	i := 0
	if len(b) > x {
		i = x
		n, err = c.Write(b[:i])
		if err != nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	n, err = c.Write(b[i:])
	n += i
	return
}

func (c *Conn) IsClosed() (bool, error) {
	_ = c.SetDeadline(time.Now().Add(100 * time.Millisecond))
	_, err := c.R.Peek(1)
	return IsClosed(err), err
}

// NewConn packs a `net.Conn` into a new `Conn`.
func NewConn(c net.Conn) *Conn {
	cc := &Conn{c, bufio.NewReader(c)}
	return cc
}

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

// ReadData is non-blocking.
func ReadData(r *bufio.Reader) ([]byte, error) {
	n := r.Buffered()
	b := make([]byte, n)
	_, err := r.Read(b)
	return b, err
}

// ReceiveData is blocking.
func ReceiveData(r *bufio.Reader) ([]byte, error) {
	_, err := r.Peek(1)
	if err != nil {
		return nil, err
	}
	return ReadData(r)
}

// ConnSolver is the interface of Conn to solve the connection prerequisites to transfer the real data.
// CONNECT to proxy. Maybe BIDN, UDP.
type ConnSolver interface {
	Bond(m, h, p string) error
	GetConn() *Conn
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
