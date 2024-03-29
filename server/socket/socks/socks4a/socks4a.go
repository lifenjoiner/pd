// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package socks4a serves as a socks4a layer server.
package socks4a

import (
	"log"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/dispatcher"
	"github.com/lifenjoiner/pd/protocol/socks"
	"github.com/lifenjoiner/pd/protocol/socks4a"
	"github.com/lifenjoiner/pd/server"
)

// Server struct.
type Server server.Server

// Serve serves 1 client.
func (s *Server) Serve(c *bufconn.Conn) bool {
	logPre := "[socks4a] " + c.RemoteAddr().String()

	req, err := socks4a.ParseRequest(c.R)
	if err != nil {
		log.Printf("%v <= %v", logPre, err)
		return false
	}

	var msg string
	switch req.Cmd {
	case socks.CONNECT:
		dp := dispatcher.New("socks4a", c, req.DestHost, req.DestPort, s.Config.UpstreamTimeout)
		dp.ParallelDial = s.Config.ParallelDial
		return dp.Dispatch(req)
	case socks.BIND:
		msg = "unimplemented BIND"
	default:
		msg = "unsupported command"
	}
	log.Printf("%v <= %v", logPre, msg)
	return false
}
