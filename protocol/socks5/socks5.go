// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package socks5

import (
	"bufio"
	"errors"
	"io"

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

func Authorize(w io.Writer, rd *bufio.Reader) (err error) {
	var p socks.Packet
	p, err = bufconn.ReceiveData(rd)
	if err != nil {
		return
	}
	if p[0] != 5 || len(p) < 3 {
		return errors.New("not SOCKS5")
	}
	// NO AUTHENTICATION REQUIRED
	_, err = w.Write([]byte{5, 0})
	return
}
