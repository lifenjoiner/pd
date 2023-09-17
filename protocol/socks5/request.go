// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package socks5

import (
	"bufio"
	"errors"
	"io"
	"time"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/forwarder"
	"github.com/lifenjoiner/pd/protocol/socks"
)

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

func (r *Request) Target() string {
	return r.DestHost + ":" + r.DestPort
}

func (r *Request) Host() string {
	return r.DestHost + ":" + r.DestPort
}

func (r *Request) Hostname() string {
	return r.DestHost
}

func (r *Request) Port() string {
	return r.DestPort
}

func (r *Request) GetRequest(w io.Writer, rd *bufio.Reader) (err error) {
	if !r.Responsed {
		_, err = w.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
		r.Responsed = true
		if err == nil {
			r.RequestData, err = bufconn.ReceiveData(rd)
		}
	}
	return
}

func (r *Request) Request(fw *forwarder.Forwarder, proxy, seg bool) (restart bool, err error) {
	_ = fw.LeftConn.SetDeadline(time.Now().Add(2 * fw.Timeout))
	_ = fw.RightConn.SetDeadline(time.Now().Add(fw.Timeout))
	if seg {
		_, err = fw.RightConn.SplitWrite(r.RequestData, 6)
	} else {
		_, err = fw.RightConn.Write(r.RequestData)
	}
	if err == nil {
		restart, err = fw.Tunnel()
	}
	return
}

func ParseRequest(rd *bufio.Reader) (req *Request, err error) {
	var p socks.Packet
	p, err = bufconn.ReceiveData(rd)
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
