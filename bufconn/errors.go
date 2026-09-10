// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package bufconn

import (
	"net"
	"strings"
)

// IsDNSErr tests if the error is `net.DNSError`.
func IsDNSErr(err error) bool {
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

// IsTimeout tests if the error is timeout.
func IsTimeout(err error) bool {
	return err != nil && strings.HasSuffix(err.Error(), "i/o timeout")
}

// IsReset tests if the error is reset.
func IsReset(err error) bool {
	return err != nil && strings.Contains(err.Error(), "forcibly closed")
}

// IsReset tests if the error means IO finished.
func IsEOF(err error) bool {
	return err != nil && err.Error() == "EOF"
}

// IsReset tests if the conn is valid.
func IsClosed(err error) bool {
	return IsReset(err) || IsEOF(err)
}
