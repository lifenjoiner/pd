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
func (s *Socks5Server) ServeSocks5(c *bufconn.Conn) bool {
	logPre := "[socks5] " + c.RemoteAddr().String()

	err := socks5.Authorize(c, c.R)
	if err != nil {
		log.Printf("%v <= %v", logPre, err)
		return false
	}
	req, err := socks5.ParseRequest(c.R)
	if err != nil {
		log.Printf("%v <= %v", logPre, err)
		return false
	}

	var msg string
	switch req.Cmd {
	case socks.CONNECT:
		return s.ServeConnect(c, req)
	case socks.BIND:
		msg = "unimplemented BIND"
	case socks.UDPASSOCIATE:
		msg = "unimplemented UDPASSOCIATE"
	default:
		msg = "unsupported command"
	}
	log.Printf("%v <= %v", logPre, msg)
	return false
}

func (s *Socks5Server) ServeConnect(client *bufconn.Conn, req *socks5.Request) bool {
	dp := dispatcher.New("socks5", client, req.DestHost, req.DestPort, s.Config.UpstreamTimeout)
	dp.ParallelDial = s.Config.ParallelDial
	return dp.Dispatch(req)
}
