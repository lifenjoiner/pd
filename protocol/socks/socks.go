// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Public socks parsing components.
package socks

import (
	"net"
	"strconv"
)

// SOCKS request commands as defined in rfc1928.
const (
	CONNECT      byte = 1
	BIND         byte = 2
	UDPASSOCIATE byte = 3
)

type Packet []byte

func (p Packet) ReadIPv4(i int) string {
	return net.IP(p[i : i+4]).String()
}

func (p Packet) ReadIPv6(i int) string {
	return net.IP(p[i : i+16]).String()
}

func (p Packet) ReadString4a(i int) (string, int) {
	j := i
	for ; j < len(p); j++ {
		if p[j] == 0 {
			break
		}
	}
	return string(p[i:j]), j
}

func (p Packet) ReadString5(i int) string {
	n := int(p[i])
	i++
	return string(p[i : i+n])
}

func (p Packet) ReadPort(i int) string {
	return strconv.Itoa((int(p[i]) << 8) | int(p[i+1]))
}

func ToPacketPort(s string) ([]byte, error) {
	var p []byte
	n, err := strconv.Atoi(s)
	if err == nil {
		p = make([]byte, 2)
		p[0] = uint8((n & 0xff00) >> 8)
		p[1] = uint8(n & 0xff)
	}
	return p, err
}
