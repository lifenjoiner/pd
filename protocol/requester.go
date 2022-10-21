// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package protocol

import (
	"bufio"
	"io"

	"github.com/lifenjoiner/pd/forwarder"
)

// The interface to send the proxied request to upstream servers.
type Requester interface {
	Command() string
	Target() string
	Host() string
	Hostname() string
	Port() string
	GetRequest(w io.Writer, r *bufio.Reader) error
	Request(fw *forwarder.Forwarder, seg bool) error
}
