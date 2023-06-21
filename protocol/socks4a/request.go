// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package socks4a implements a socks4a proxy.
package socks4a

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
	Ver byte
	Cmd byte
	//AddrType    byte
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
		_, err = w.Write([]byte{0, 0x5a, 0, 0, 0, 0, 0, 0})
		r.Responsed = true
		if err == nil {
			r.RequestData, err = bufconn.ReceiveData(rd)
		}
	}
	return
}

func (r *Request) Request(fw *forwarder.Forwarder, seg bool) (restart bool, err error) {
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

	// no authorization
	if p[0] != 4 || len(p) < 10 {
		return nil, errors.New("not SOCKS4a")
	}

	req = &Request{}
	req.PacketData = p
	req.Ver = p[0]
	req.Cmd = p[1]
	req.DestPort = p.ReadPort(2)
	if p[4] > 0 {
		req.DestHost = p.ReadIPv4(4)
	} else {
		_, i := p.ReadString4a(8)
		req.DestHost, _ = p.ReadString4a(i + 1)
	}
	return
}
