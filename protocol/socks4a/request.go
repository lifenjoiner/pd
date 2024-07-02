// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package socks4a offers protocol operations.
package socks4a

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"time"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/forwarder"
	"github.com/lifenjoiner/pd/protocol/socks"
)

// Request struct.
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

// Command requested by.
func (r *Request) Command() (m string) {
	switch r.Cmd {
	case CmdConnect:
		m = "CONNECT"
	case CmdBind:
		m = "BIND"
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

// GetRequest requests the ClientHello for sending to a remote server.
// RCWN (Race Cache With Network) or ads blockers would abort dial-in without sendig ClientHello! Drop it.
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

// Request to a upstream server.
func (r *Request) Request(fw *forwarder.Forwarder, _, seg bool) (restart bool, err error) {
	_ = fw.LeftConn.SetDeadline(time.Now().Add(2 * fw.Timeout))
	_ = fw.RightConn.SetDeadline(time.Now().Add(fw.Timeout))
	if seg {
		i := bytes.Index(r.RequestData, []byte(r.DestHost))
		i += len(r.DestHost) / 2
		_, err = fw.RightConn.SplitWrite(r.RequestData, i)
	} else {
		_, err = fw.RightConn.Write(r.RequestData)
	}
	if err == nil {
		restart, err = fw.Tunnel()
	}
	return
}

// ParseRequest parses a request.
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
