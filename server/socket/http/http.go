// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package http serves as http layer server.
package http

import (
	"log"
	"os"

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
		if len(s.Config.PacFile) > 0 && len(u.Path) > 1 && u.Path[0] == '/' && u.Path[1:] == s.Config.PacFile {
			return s.servePac(c)
		}
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

func (s *Server) servePac(c *bufconn.Conn) bool {
	log.Printf("[http] pac: %v <- %v", s.Config.PacFile, c.RemoteAddr())
	b, err := os.ReadFile(s.Config.PacFile)
	if err == nil {
		_, err = c.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: application/x-ns-proxy-autoconfig\r\nConnection: close\r\n\r\n"))
		if err == nil {
			_, err = c.Write(b)
			if err == nil {
				return true
			}
		}
	}
	log.Printf("[http] Pac file: %v", err)
	return false
}
