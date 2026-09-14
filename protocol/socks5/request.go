// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package socks5

import (
	"bytes"
	"errors"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/protocol/socks"
)

// Request struct.
type Request struct {
	Ver         byte
	Cmd         byte
	AddrType    byte
	DestHost    string
	DestPort    string
	PacketData  socks.Packet
	RequestData socks.Packet
	Responsed   bool
}

// Command requested by.
func (r *Request) Command() (m string) {
	switch r.Cmd {
	case CmdConnect:
		m = "CONNECT"
	case CmdBind:
		m = "BIND"
	case CmdUDP:
		m = "UDP"
	}
	return
}

// Target URL requested to.
func (r *Request) Target() string {
	return r.DestHost + ":" + r.DestPort
}

// Host requested to.
func (r *Request) Host() string {
	return r.DestHost + ":" + r.DestPort
}

// Hostname only requested to.
func (r *Request) Hostname() string {
	return r.DestHost
}

// Port requested to.
func (r *Request) Port() string {
	return r.DestPort
}

// GetInnerRequest requests the ClientHello for sending to a remote server.
// RCWN (Race Cache With Network) or ads blockers would abort dial-in without sendig ClientHello! Drop it.
func (r *Request) GetInnerRequest(c *bufconn.Conn) (err error) {
	if !r.Responsed {
		_, err = c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
		r.Responsed = true
		if err == nil {
			r.RequestData, err = c.ReadAll()
		}
	}
	return
}

// Request to a upstream server.
func (r *Request) Request(c *bufconn.Conn, _, seg bool) (err error) {
	if seg {
		i := bytes.Index(r.RequestData, []byte(r.DestHost))
		i += len(r.DestHost) / 2
		_, err = c.SplitWrite(r.RequestData, i)
	} else {
		_, err = c.Write(r.RequestData)
	}
	return
}

// ParseRequest parses a request.
func ParseRequest(c *bufconn.Conn) (req *Request, err error) {
	var p socks.Packet
	p, err = c.ReadAll()
	if err != nil {
		return
	}
	if len(p) < 4 {
		return nil, errors.New("illegal SOCKS5 packet")
	}
	req = &Request{}
	req.PacketData = p
	req.Ver = p[0]
	req.Cmd = p[1]
	req.AddrType = p[3]
	i := 4
	n := 0
	switch req.AddrType {
	case ATypeIPv4:
		n = 4
	case ATypeIPv6:
		n = 16
	case ATypeDomain:
		n = int(p[i]) + 1
	default:
		return nil, errors.New("invalid SOCKS5 address type")
	}
	if len(p) < 4+n+2 {
		return nil, errors.New("illegal SOCKS5 packet")
	}
	switch req.AddrType {
	case ATypeIPv4:
		req.DestHost = p.ReadIPv4(4)
	case ATypeIPv6:
		req.DestHost = p.ReadIPv6(4)
	case ATypeDomain:
		req.DestHost = p.ReadString5(4)
	}
	i += n
	req.DestPort = p.ReadPort(i)
	return
}
