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

type HttpConn Conn

func (c *HttpConn) bondData(m, h, p string) ([]byte, error) {
	switch strings.ToUpper(m) {
	case "CONNECT":
	default:
		return nil, nil
	}

	hp := net.JoinHostPort(h, p)
	data := []byte("CONNECT " + hp + " HTTP/1.1\r\nHost: " + hp + "\r\n\r\n")
	return data, nil
}

func (c *HttpConn) Bond(m, h, p string, b []byte) (err error) {
	if len(b) == 0 {
		b, err = c.bondData(m, h, p)
	}
	if len(b) == 0 {
		return
	}
	_, err = c.Write(b)
	if err == nil {
		var line []byte
		line, _, err = c.R.ReadLine()
		if err == nil {
			if strings.Contains(string(line), " 200 ") {
				_, err = c.R.Discard(c.R.Buffered())
				return
			}
			err = errors.New("[http] not available proxy server")
		}
	}
	return err
}

func (c *HttpConn) GetConn() *Conn {
	return (*Conn)(c)
}

func NewHttpConn(c *Conn) *HttpConn {
	return (*HttpConn)(c)
}

func DialHttp(u *url.URL, d time.Duration) (*HttpConn, error) {
	c, err := DialURL(u, d)
	return (*HttpConn)(c), err
}
