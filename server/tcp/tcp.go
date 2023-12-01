// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package tcp serves as a TCP layer server.
package tcp

import (
	"log"
	"net"
	"time"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/server"
	"github.com/lifenjoiner/pd/server/socket/http"
	"github.com/lifenjoiner/pd/server/socket/socks/socks4a"
	"github.com/lifenjoiner/pd/server/socket/socks/socks5"
)

// Server stores the socks/http proxy config.
type Server server.Server

// ListenAndServe listens on the Addr and serves connections.
func (s *Server) ListenAndServe() {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		log.Printf("[tcp] failed to listen on %s: %v\n", s.Addr, err)
		return
	}
	defer l.Close()

	log.Printf("[tcp] listening on %s\n", s.Addr)
	for {
		c, err := l.Accept()
		if err != nil {
			log.Printf("[tcp] failed to accept: %v\n", err)
			continue
		}
		cc := bufconn.NewConn(c)
		go s.Serve(cc)
	}
}

// Serve serves 1 client.
func (s *Server) Serve(c *bufconn.Conn) {
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(2 * s.Config.UpstreamTimeout))

	data, err := c.R.Peek(1)
	if err != nil {
		log.Printf("[tcp] drop %v, error: %v", c.RemoteAddr(), err)
		return
	}
	switch data[0] {
	case 5:
		socks5 := (*socks5.Server)(s)
		socks5.Serve(c)
	case 4:
		socks4a := (*socks4a.Server)(s)
		socks4a.Serve(c)
	default:
		http := (*http.Server)(s)
		http.Serve(c)
	}
}
