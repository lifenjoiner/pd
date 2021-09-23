// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Socks4a layer server entry.
package socks4a

import (
	"log"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/dispatcher"
	"github.com/lifenjoiner/pd/protocol/socks"
	"github.com/lifenjoiner/pd/protocol/socks4a"
	"github.com/lifenjoiner/pd/server"
)

type Socks4aServer server.Server

// Serve 1 client.
func (s *Socks4aServer) ServeSocks4a(c *bufconn.Conn) {
	logPre := "[socks4a] " + c.RemoteAddr().String()

	req, err := socks4a.ParseRequest(c.R)
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
	default:
		msg = "unsupported command"
	}
	log.Printf("%v <= %v", logPre, msg)
}

func (s *Socks4aServer) ServeConnect(client *bufconn.Conn, req *socks4a.Request) {
	dp := &dispatcher.Dispatcher{
		ServerType: "socks4a",
		Client:     client,
		DestHost:   req.DestHost,
		DestPort:   req.DestPort,
		Timeout:    s.Config.UpstreamTimeout,
	}
	dp.Dispatch(req)
}
