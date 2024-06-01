// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package checker is a URL based remote server availability checher.
package checker

import (
	"errors"
	"net/url"
	"time"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/protocol"
)

// TargetChecker is the struct to check an URL directly or through proxy.
type TargetChecker struct {
	*url.URL
	Timeout time.Duration
	Conn    *bufconn.Conn
	Proxied *url.URL
}

// Transfer operates the check communication.
func (ck *TargetChecker) Transfer() (err error) {
	conn := ck.Conn
	u := ck.Proxied
	if u == nil {
		u = ck.URL
	}
	_ = conn.SetDeadline(time.Now().Add(ck.Timeout))
	switch u.Scheme {
	case "https":
		_, err = conn.Write([]byte("\x15\x03\x03\x00\x01\x00"))
	case "http":
		_, err = conn.Write([]byte("HEAD / HTTP/1.1\r\nHost: " + ck.Host + "\r\n\r\n"))
	default:
		err = errors.New("[TargetChecker] unkown scheme: " + u.Scheme)
	}
	if err != nil {
		return
	}
	_, err = conn.R.ReadByte()
	if err == nil {
		_, err = conn.R.Discard(conn.R.Buffered())
	} else {
		err = errors.New("[TargetChecker] no response")
	}
	return
}

// Check checks if the target responses succeeded.
func (ck *TargetChecker) Check() (err error) {
	var cs bufconn.ConnSolver
	switch ck.URL.Scheme {
	case "http", "https":
		cs, err = bufconn.DialHTTP(ck.URL, ck.Timeout)
	case "socks5":
		cs, err = bufconn.DialSocks5(ck.URL, ck.Timeout)
	case "socks4a":
		cs, err = bufconn.DialSocks4a(ck.URL, ck.Timeout)
	default:
		err = errors.New("[TargetChecker] unknown scheme: " + ck.URL.Scheme)
		return
	}
	if err == nil {
		conn := cs.GetConn()
		_ = conn.SetDeadline(time.Now().Add(ck.Timeout))
		pu := ck.Proxied
		if ck.Proxied != nil {
			port := protocol.GetPort(pu)
			if len(port) > 0 {
				err = cs.Bond("CONNECT", pu.Hostname(), port, nil)
			} else {
				err = errors.New("[TargetChecker] unknown port for target")
			}
		}
		if err == nil {
			ck.Conn = conn
			err = ck.Transfer()
		}
		_ = conn.SetDeadline(time.Now())
		conn.Close()
	}
	return
}

// NewTargetChecker packs a new TargetChecker.
func NewTargetChecker(u *url.URL, d time.Duration, c *bufconn.Conn, p *url.URL) *TargetChecker {
	return &TargetChecker{u, d, c, p}
}

// New generates a new TargetChecker from a URL string.
func New(s string, d time.Duration, p string) (*TargetChecker, error) {
	if len(s) == 0 {
		return nil, errors.New("[TargetChecker] server URL is empty")
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, errors.New("[TargetChecker] server URL is invalid")
	}
	var pu *url.URL
	if len(p) > 0 {
		pu, err = url.Parse(p)
	}
	if err != nil {
		return nil, errors.New("[TargetChecker] proxy URL is invalid")
	}
	return NewTargetChecker(u, d, nil, pu), nil
}
