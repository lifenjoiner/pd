// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package protocol

import (
	"net/url"
	"strings"
)

// Get the omitted port of an URL.
func GetPort(u *url.URL) string {
	port := u.Port()
	if len(port) == 0 {
		switch strings.ToLower(u.Scheme) {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}
	return port
}
