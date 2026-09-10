// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package bufconn offers access to connections with buffer.
package bufconn

import (
	"bufio"
	"net"
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
