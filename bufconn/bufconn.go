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

// Conn is a deadline sensitive connection with bufio reader.
// Set the deadline before each IO. Always timeout here.
type Conn struct {
	C       net.Conn
	R       *bufio.Reader
	Timeout time.Duration
}

// SetReadDeadline sets the read deadline for the connection.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.C.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline for the connection.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.C.SetWriteDeadline(t)
}

// SetDeadline sets the read and write deadlines for the connection.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.C.SetDeadline(t)
}

// ReadTimeout reads data with specified timeout.
func (c *Conn) ReadTimeout(b []byte, d time.Duration) (int, error) {
	_ = c.C.SetReadDeadline(time.Now().Add(d))
	return c.C.Read(b)
}

// Read is blocking with default timeout.
func (c *Conn) Read(b []byte) (int, error) {
	return c.ReadTimeout(b, c.Timeout)
}

// ReadAllTimeout reads all data with specified timeout.
func (c *Conn) ReadAllTimeout(d time.Duration) ([]byte, error) {
	_ = c.C.SetReadDeadline(time.Now().Add(d))
	return ReceiveData(c.R)
}

// ReadAll is blocking with default timeout.
func (c *Conn) ReadAll() ([]byte, error) {
	return c.ReadAllTimeout(c.Timeout)
}

// PeekTimeout peeks data with specified timeout.
func (c *Conn) PeekTimeout(n int, d time.Duration) ([]byte, error) {
	_ = c.C.SetReadDeadline(time.Now().Add(d))
	return c.R.Peek(n)
}

// Peek peeks data with default timeout.
func (c *Conn) Peek(n int) ([]byte, error) {
	return c.PeekTimeout(n, c.Timeout)
}

// WriteTimeout writes data with specified timeout.
func (c *Conn) WriteTimeout(b []byte, d time.Duration) (int, error) {
	_ = c.C.SetWriteDeadline(time.Now().Add(d))
	return c.C.Write(b)
}

// Write writes data with default timeout.
func (c *Conn) Write(b []byte) (int, error) {
	return c.WriteTimeout(b, c.Timeout)
}

// SplitWriteTimeout is to break length pattern, and writes with specified timeout.
func (c *Conn) SplitWriteTimeout(b []byte, x int, d time.Duration) (n int, err error) {
	i := 0
	if len(b) > x {
		i = x
		n, err = c.WriteTimeout(b[:i], d)
		if err != nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	n, err = c.WriteTimeout(b[i:], d)
	n += i
	return
}

// SplitWrite is to break length pattern, and writes with default timeout.
func (c *Conn) SplitWrite(b []byte, x int) (n int, err error) {
	return c.SplitWriteTimeout(b, x, c.Timeout)
}

// Close closes the connection now.
func (c *Conn) Close() error {
	_ = c.C.SetDeadline(time.Now())
	return c.C.Close()
}

// IsClosed returns if the connection is closed and reason.
func (c *Conn) IsClosed() (bool, error) {
	_ = c.SetDeadline(time.Now().Add(time.Millisecond))
	_, err := c.R.Peek(1)
	return IsClosed(err), err
}

// LocalAddr wraps same name function of net.Conn.
func (c *Conn) LocalAddr() net.Addr {
	return c.C.LocalAddr()
}

// RemoteAddr wraps same name function of net.Conn.
func (c *Conn) RemoteAddr() net.Addr {
	return c.C.RemoteAddr()
}

// NewConn packs a `net.Conn` into a new `Conn`.
func NewConn(c net.Conn, d time.Duration) *Conn {
	cc := &Conn{c, bufio.NewReader(c), d}
	return cc
}

// ReadAll is non-blocking.
func ReadAll(r *bufio.Reader) ([]byte, error) {
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
	return ReadAll(r)
}

// ConnSolver is the interface of Conn to solve the connection prerequisites to transfer the real data.
// CONNECT to proxy. Maybe BIND, UDP.
type ConnSolver interface {
	Bond(m, h, p string) error
	GetConn() *Conn
}
