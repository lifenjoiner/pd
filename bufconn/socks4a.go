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

type Socks4aConn Conn

func (c *Socks4aConn) bondData(m, h, p string) ([]byte, error) {
	switch strings.ToUpper(m) {
	case "CONNECT":
	default:
		return nil, errors.New("unsupported method: "+ m)
	}

	pp, err := socks.ToPacketPort(p)
	if err != nil {
		return nil, err
	}
	l := len(h)
	if l > 256 {
		return nil, errors.New("too long hostname: "+ h)
	}
	data := []byte{4, 1}
	data = append(data, pp...)
	data = append(data, []byte{0, 0, 0, 1}...)
	data = append(data, byte(0))
	data = append(data, []byte(h)...)
	data = append(data, byte(0))
	return data, err
}

func (c *Socks4aConn) Bond(m, h, p string, b []byte) (err error) {
	if len(b) == 0 {
		b, err = c.bondData(m, h, p)
	}
	if len(b) == 0 {
		return
	}
	_, err = c.Write(b)
	if err == nil {
		var b socks.Packet
		b, err = ReceiveData(c.R)
		if err == nil {
			if b[1] == 0x5a {
				return nil
			}
			err = errors.New("CONNECT failed")
		}
	}
	return err
}

func (c *Socks4aConn) GetConn() (*Conn) {
	return (*Conn)(c)
}

func NewSocks4aConn(c *Conn, u *url.URL) (*Socks4aConn) {
	return (*Socks4aConn)(c)
}

func DialSocks4a(u *url.URL, d time.Duration) (*Socks4aConn, error) {
	c, err := DialURL(u, d)
	return (*Socks4aConn)(c), err
}
