// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package http offers protocol operations.
package http

import (
	"github.com/lifenjoiner/pd/bufconn"
)

// Authorize a client permission to proceed. Dummy.
func Authorize(c *bufconn.Conn) (err error) {
	return
}
