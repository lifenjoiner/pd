// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package forwarder

import (
	"strings"
)

func isTimeout(err error) bool {
	return err != nil && strings.HasSuffix(err.Error(), "i/o timeout")
}

func isReset(err error) bool {
	return err != nil && strings.Contains(err.Error(), "forcibly closed")
}

func isEOF(err error) bool {
	return err != nil && err.Error() == "EOF"
}
