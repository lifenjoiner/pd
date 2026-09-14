// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package socks5 offers protocol operations.
package socks5

import (
	"errors"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/protocol/socks"
)

// SOCKS auth type.
const (
	AUTHNONE     byte = 0
	AUTHPASSWORD byte = 2
)

// SOCKS address types.
const (
	CmdConnect = 1
	CmdBind    = 2
	CmdUDP     = 3
)

// SOCKS address types.
const (
	ATypeIPv4   = 1
	ATypeDomain = 3
	ATypeIPv6   = 4
)

// Authorize a client permission to proceed.
func Authorize(c *bufconn.Conn) (err error) {
	var p socks.Packet
	p, err = c.ReadAll()
	if err != nil {
		return
	}
	if p[0] != 5 || len(p) < 3 {
		return errors.New("not SOCKS5")
	}
	// NO AUTHENTICATION REQUIRED
	_, err = c.Write([]byte{5, 0})
	return
}
