// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package bufconn

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/lifenjoiner/pd/protocol/socks"
)

type Socks5Conn Conn

func (c *Socks5Conn) authorize() (err error) {
	_, err = c.Write([]byte{5, 1, 0})
	if err != nil {
		return
	}
	var p socks.Packet
	p, err = ReceiveData(c.R)
	if err == nil && p[1] != 0 {
		err = errors.New("[socks5] authorization failed")
	}
	return
}

func (c *Socks5Conn) bondData(m, h, p string) ([]byte, error) {
	switch strings.ToUpper(m) {
	case "CONNECT":
	default:
		return nil, errors.New("[socks5] unsupported method: " + m)
	}

	pp, err := socks.ToPacketPort(p)
	if err != nil {
		return nil, err
	}
	l := len(h)
	if l > 256 {
		return nil, errors.New("[socks5] too long hostname: " + h)
	}
	data := []byte{5, 1, 0, 3}
	data = append(data, byte(l))
	data = append(data, []byte(h)...)
	data = append(data, pp...)
	return data, err
}

func (c *Socks5Conn) Bond(m, h, p string, b []byte) (err error) {
	if len(b) == 0 {
		b, err = c.bondData(m, h, p)
	}
	if len(b) == 0 {
		return
	}
	err = c.authorize()
	if err == nil {
		_, err = c.Write(b)
		if err == nil {
			var b socks.Packet
			b, err = ReceiveData(c.R)
			if err == nil {
				if b[1] == 0 {
					return nil
				}
				err = errors.New("[socks5] CONNECT failed")
			}
		}
	}
	return
}

func (c *Socks5Conn) GetConn() *Conn {
	return (*Conn)(c)
}

func NewSocks5Conn(c *Conn, u *url.URL) *Socks5Conn {
	return (*Socks5Conn)(c)
}

func DialSocks5(u *url.URL, d time.Duration) (*Socks5Conn, error) {
	c, err := DialURL(u, d)
	return (*Socks5Conn)(c), err
}
