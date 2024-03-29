// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package http serves as http layer server.
package http

import (
	"log"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/dispatcher"
	"github.com/lifenjoiner/pd/protocol/http"
	"github.com/lifenjoiner/pd/server"
)

// Server struct.
type Server server.Server

// Serve serves 1 client.
func (s *Server) Serve(c *bufconn.Conn) bool {
	req, err := http.ParseRequest(c.R)
	if err != nil {
		log.Printf("[http] %v", err)
		return false
	}

	u := req.URL
	if u.Host == "" {
		log.Printf("[http] Invalid request.")
		return false
	}

	dp := dispatcher.New("http", c, u.Hostname(), u.Port(), s.Config.UpstreamTimeout)
	if dp.DestPort == "" && req.Method != "CONNECT" {
		if u.Scheme == "" {
			u.Scheme = "http"
		}
		dp.DestPort = u.Scheme
	}
	dp.ParallelDial = s.Config.ParallelDial
	return dp.Dispatch(req)
}
