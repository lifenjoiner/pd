// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package bufconn offers access to connections with buffer.
package bufconn

import (
	"net"
	"syscall"
)

func bindToDevice(ifi *net.Interface) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		return c.Control(func(fd uintptr) {
			if ifi != nil {
				syscall.BindToDevice(int(fd), ifi.Name)
			}
		})
	}
}
