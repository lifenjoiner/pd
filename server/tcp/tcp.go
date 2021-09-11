// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package tcp

import (
	"bytes"
	"log"
	"net"
	"time"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/server"
	"github.com/lifenjoiner/pd/server/socket/http"
	"github.com/lifenjoiner/pd/server/socket/socks/socks4a"
	"github.com/lifenjoiner/pd/server/socket/socks/socks5"
)

// TCPServer stores the socks/http proxy config.
type TCPServer server.Server

// ListenAndServe listens on the Addr and serves connections.
func (s *TCPServer) ListenAndServe() {
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
func (s *TCPServer) Serve(c *bufconn.Conn) {
	defer c.Close()
	c.SetDeadline(time.Now().Add(2 * s.Config.UpstreamTimeout))

	data, err := c.R.Peek(1)
	if err != nil {
		log.Printf("[tcp] drop %v, error: %v", c.RemoteAddr(), err)
		return
	}
	switch data[0] {
	case 5:
		socks5 := (*socks5.Socks5Server)(s)
		socks5.ServeSocks5(c)
		return
	case 4:
		socks4a := (*socks4a.Socks4aServer)(s)
		socks4a.ServeSocks4a(c)
		return
	}

	data, err = c.R.Peek(8)
	if err != nil {
		log.Printf("[tcp] drop %v, error: %v", c.RemoteAddr(), err)
		return
	}
	if bytes.Equal(data, []byte("CONNECT ")) || bytes.HasPrefix(data, []byte("GET ")) {
		http := (*http.HttpServer)(s)
		http.ServeHttp(c)
		return
	}
}
