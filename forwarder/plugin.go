// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package forwarder

type Transformer interface {
	Transform([]byte) []byte
}
