// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// A host (ip) matching model.
package statichost

import (
	"log"
	"os"
	"strings"
)

// The strategies the hosts will be processed with.
const (
	StaticNil = iota
	StaticDirect
	StaticBlocked
)

type Strategy byte

type StaticHosts map[string]Strategy

// Load settings from a file.
func (sh StaticHosts) Load(file string, strategy Strategy) {
	data, err := os.ReadFile(file)
	if err != nil {
		log.Printf("[hoststats] %v", err)
	}
	sh.Upsert(string(data), strategy)
}

// Update/insert the StaticHosts by line(s) of items for a same strategy.
// host: sufix, ip: prefix.
func (sh StaticHosts) Upsert(in string, strategy Strategy) {
	lines := strings.Split(in, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		dm := strings.Fields(line)[0]
		if dm[0] == '#' {
			continue
		}
		sh[dm] = strategy
	}
}

// Get the strategy of an hostname. Right to left, match sufix after the separator first.
func (sh StaticHosts) GetHostStrategy(host string) Strategy {
	h := "." + host
	for i := len(host); i >= 0; i-- {
		if h[i] != '.' {
			continue
		}
		dv := sh[h[i+1:]]
		if dv != StaticNil {
			return dv
		}
	}
	return StaticNil
}

// Get the strategy of an ip. Left to right.
// IP syntax: a.b.c.d, 127.0.0.*, 192.168.*, or 10.*; * is required as IPv6 would omit `0`s.
func (sh StaticHosts) GetIPStrategy(ip string) Strategy {
	// "::ffff:192.0.2.1"
	var sp byte = ':'
	if strings.LastIndexByte(ip, '.') > 0 {
		sp = '.'
	}
	for i := 0; i < len(ip); i++ {
		if ip[i] != sp {
			continue
		}
		dv := sh[ip[0:i+1]+"*"]
		if dv != StaticNil {
			return dv
		}
	}
	return sh[ip]
}

// Get the strategy for a host or ip.
func (sh StaticHosts) GetStrategy(q string) Strategy {
	if HostIsIP(q) {
		return sh.GetIPStrategy(q)
	} else {
		return sh.GetHostStrategy(q)
	}
}

func HostIsIP(h string) bool {
	n := len(h)
	if n <= 0 {
		return false
	}
	v := h[n-1]
	return '0' <= v && v <= '9' || strings.ContainsRune(h, ':')
}

// Load all settings.
// Priority: StaticDirect > StaticBlocked
func MapStaticFiles(blocked, direct string) StaticHosts {
	sh := StaticHosts{}
	sh.Load(blocked, StaticBlocked)
	sh.Load(direct, StaticDirect)
	return sh
}
