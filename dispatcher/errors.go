// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package dispatcher

import (
	"net"
)

func IsDnsErr(err error) (bool) {
	if err == nil {
		return false
	}
	_, ok := err.(*net.DNSError)
	if ok {
		return true
	}
	opErr, ok2 := err.(*net.OpError)
	if ok2 {
		_, ok2 = opErr.Err.(*net.DNSError)
	}
	return ok2
}
