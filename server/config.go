// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package server

import (
	"time"
)

type Config struct {
	UpstreamTimeout time.Duration
	ParallelDial    bool
	Proxies         string
	ProxyProbeURL   string
}
