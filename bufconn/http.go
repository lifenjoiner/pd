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
func (c *HTTPConn) Bond(m, h, p string, b []byte) (err error) {
	if len(b) == 0 {
		b, err = c.bondData(m, h, p)
	}
	if len(b) == 0 {
		return
	}
	_, err = c.Write(b)
	if err == nil {
		var line string
		var ok, eoh bool
		// cover fragmentations by http.Response.Write(). c has timeout.
		wait := 3
		for i := 0; i < wait+1; {
			if !eoh || c.R.Buffered() > 0 {
				line, err = c.R.ReadString('\n')
				if i == 0 {
					i++
					ok = strings.Contains(line, " 200 ")
				} else if err != nil {
					i++
				} else if !eoh {
					eoh = line == "\r\n"
				}
			} else {
				i++
				time.Sleep(time.Millisecond)
			}
		}
		if !ok {
			err = errors.New("http proxy server: not available")
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
