// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// HTTP layer server entry.
package http

import (
	"log"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/dispatcher"
	"github.com/lifenjoiner/pd/protocol/http"
	"github.com/lifenjoiner/pd/server"
)

type HttpServer server.Server

// Serve 1 client.
func (s *HttpServer) ServeHttp(c *bufconn.Conn) {
	req, err := http.ParseRequest(c.R)
	if err != nil {
		log.Printf("[http] ", err)
		return
	}

	u := req.URL
	isGet := req.Method == "GET"
	isConnect := req.Method == "CONNECT"
	if (isGet && !u.IsAbs()) || (isConnect && u.Host == "") {
		log.Printf("[http] Invalid request.")
		return
	}

	dp := dispatcher.New("http", c, u.Hostname(), u.Port(), s.Config.UpstreamTimeout)
	if isGet && dp.DestPort == "" {
		if u.Scheme == "" {
			u.Scheme = "http"
		}
		dp.DestPort = u.Scheme
	}
	dp.ParallelDial = s.Config.ParallelDial
	dp.Dispatch(req)
}
