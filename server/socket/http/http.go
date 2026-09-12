// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package http serves as http layer server.
package http

import (
	"log"
	"os"
	"time"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/dispatcher"
	"github.com/lifenjoiner/pd/protocol/http"
	"github.com/lifenjoiner/pd/server"
)

// Server struct.
type Server server.Server

// Serve serves 1 client.
func (s *Server) Serve(c *bufconn.Conn) bool {
	sc := s.Config
	_ = c.SetReadDeadline(time.Now().Add(c.Timeout))
	req, err := http.ParseRequest(c.R)
	if err != nil {
		log.Printf("[http] %v", err)
		return false
	}

	u := req.URL
	if u.Host == "" {
		if len(sc.PacFile) > 0 && len(u.Path) > 1 && u.Path[0] == '/' && u.Path[1:] == sc.PacFile {
			return s.servePac(c)
		}
		log.Printf("[http] Invalid request.")
		return false
	}

	dp := dispatcher.New("http", c, u.Hostname(), u.Port(), sc.UpstreamTimeout)
	if dp.DestPort == "" && req.Method != "CONNECT" {
		if u.Scheme == "" {
			u.Scheme = "http"
		}
		dp.DestPort = u.Scheme
	}
	dp.ParallelDial = sc.ParallelDial
	return dp.Dispatch(req)
}

func (s *Server) servePac(c *bufconn.Conn) bool {
	sc := s.Config
	log.Printf("[http] pac: %v <- %v", sc.PacFile, c.RemoteAddr())
	b, err := os.ReadFile(sc.PacFile)
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
