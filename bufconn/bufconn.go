// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package bufconn

import (
	"bufio"
	"net"
	"net/url"
	"time"
)

// Conn is a connection with bufio reader.
type Conn struct {
	net.Conn
	R *bufio.Reader
}

// Non-blocking.
func (c *Conn) ReadData() ([]byte, error) {
	return ReadData(c.R)
}

// Blocking.
func (c *Conn) ReceiveData() ([]byte, error) {
	return ReceiveData(c.R)
}

// Break length pattern.
func (c *Conn) SplitWrite(b []byte, x int) (n int, err error) {
	i := 0
	if len(b) > x {
		i = x
		n, err = c.Write(b[:i])
		if err != nil {
			return
		} else {
			time.Sleep(time.Millisecond)
		}
	}
	n, err = c.Write(b[i:])
	n += i
	return
}

func NewConn(c net.Conn) *Conn {
	cc := &Conn{c, bufio.NewReader(c)}
	return cc
}

func Dial(network, address string, timeout time.Duration) (*Conn, error) {
	c, err := net.DialTimeout(network, address, timeout)
	var conn *Conn
	if err == nil {
		conn = NewConn(c)
	}
	return conn, err
}

func DialURL(u *url.URL, d time.Duration) (*Conn, error) {
	a := u.Host
	if len(u.Port()) == 0 {
		a += ":" + u.Scheme
	}
	n := "tcp"
	if u.Scheme == "h3" {
		n = "udp"
	}
	return Dial(n, a, d)
}

// Non-blocking.
func ReadData(r *bufio.Reader) ([]byte, error) {
	n := r.Buffered()
	b := make([]byte, n)
	_, err := r.Read(b)
	return b, err
}

// Blocking.
func ReceiveData(r *bufio.Reader) ([]byte, error) {
	_, err := r.Peek(1)
	if err != nil {
		return nil, err
	}
	return ReadData(r)
}

// The interface of Conn to solve the connection prerequisites to transfer the real data.
// CONNECT to proxy. Maybe BIDN, UDP.
type ConnSolver interface {
	Bond(m, h, p string, b []byte) error
	GetConn() *Conn
}
