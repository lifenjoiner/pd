// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Socks5 layer server entry.
package socks5

import (
	"log"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/dispatcher"
	"github.com/lifenjoiner/pd/protocol/socks"
	"github.com/lifenjoiner/pd/protocol/socks5"
	"github.com/lifenjoiner/pd/server"
)

type Socks5Server server.Server

// Serve 1 client.
func (s *Socks5Server) ServeSocks5(c *bufconn.Conn) {
	logPre := "[socks5] " + c.RemoteAddr().String()

	err := socks5.Authorize(c, c.R)
	if err != nil {
		log.Printf("%v <= %v", logPre, err)
		return
	}
	req, err := socks5.ParseRequest(c.R)
	if err != nil {
		log.Printf("%v <= %v", logPre, err)
		return
	}

	var msg string
	switch req.Cmd {
	case socks.CONNECT:
		s.ServeConnect(c, req)
		return
	case socks.BIND:
		msg = "unimplemented BIND"
	case socks.UDPASSOCIATE:
		msg = "unimplemented UDPASSOCIATE"
	default:
		msg = "unsupported command"
	}
	log.Printf("%v <= %v", logPre, msg)
}

func (s *Socks5Server) ServeConnect(client *bufconn.Conn, req *socks5.Request) {
	dp := &dispatcher.Dispatcher{
		ServerType: "socks5",
		Client:     client,
		DestHost:   req.DestHost,
		DestPort:   req.DestPort,
		Timeout:    s.Config.UpstreamTimeout,
	}
	dp.Dispatch(req)
}
