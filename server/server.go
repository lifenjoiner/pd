// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// The pd server.
package server

// Server stores the pd server config.
type Server struct {
	Addr    string
	Config  *Config
}
