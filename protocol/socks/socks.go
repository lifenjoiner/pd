// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package socks offers socks parsing operations.
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

// Packet type represents the raw data.
type Packet []byte

// ReadIPv4 reads a IPv4 string from the packet.
func (p Packet) ReadIPv4(i int) string {
	return net.IP(p[i : i+4]).String()
}

// ReadIPv6 reads a IPv6 string from the packet.
func (p Packet) ReadIPv6(i int) string {
	return net.IP(p[i : i+16]).String()
}

// ReadString4a reads a string from the socket4a packet.
func (p Packet) ReadString4a(i int) (string, int) {
	j := i
	for ; j < len(p); j++ {
		if p[j] == 0 {
			break
		}
	}
	return string(p[i:j]), j
}

// ReadString5 reads a string from the socket5 packet.
func (p Packet) ReadString5(i int) string {
	n := int(p[i])
	i++
	return string(p[i : i+n])
}

// ReadPort reads the port into a string.
func (p Packet) ReadPort(i int) string {
	return strconv.Itoa((int(p[i]) << 8) | int(p[i+1]))
}

// ToPacketPort converts a port string to integer in packet byte order.
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
