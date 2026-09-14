// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package bufconn

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

// HTTPConn represents a HTTP connection.
type HTTPConn Conn

func (c *HTTPConn) bondData(m, h, p string) ([]byte, error) {
	switch strings.ToUpper(m) {
	case "CONNECT":
	default:
		return nil, nil
	}

	hp := net.JoinHostPort(h, p)
	data := []byte("CONNECT " + hp + " HTTP/1.1\r\nHost: " + hp + "\r\n\r\n")
	return data, nil
}

// Bond bonds a HTTP connection with the server.
func (c *HTTPConn) Bond(m, h, p string) error {
	b, err := c.bondData(m, h, p)
	if len(b) == 0 {
		return err
	}
	cc := c.GetConn()
	_, err = cc.Write(b)
	if err == nil {
		b, err = cc.ReadAll()
		if err == nil {
			s := string(b)
			if !strings.Contains(s, " 200 ") || strings.LastIndex(s, "\r\n\r\n") == -1 {
				err = errors.New("http proxy server: not available")
			}
		}
	}
	return err
}

// GetConn returns the packed `*Conn` from a `*HTTPConn`.
func (c *HTTPConn) GetConn() *Conn {
	return (*Conn)(c)
}

// DialHTTP dials a HTTP URL with timeout.
func DialHTTP(u *url.URL, d time.Duration) (*HTTPConn, error) {
	c, err := DialURL(u, d)
	return (*HTTPConn)(c), err
}
