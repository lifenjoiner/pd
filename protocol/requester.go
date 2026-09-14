// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package protocol offers operations for protocols.
package protocol

import (
	"github.com/lifenjoiner/pd/bufconn"
)

// Requester sends the proxied request to upstream servers.
type Requester interface {
	Command() string
	Target() string
	Host() string
	Hostname() string
	Port() string
	GetInnerRequest(c *bufconn.Conn) error
	Request(c *bufconn.Conn, proxy, seg bool) (err error)
}
