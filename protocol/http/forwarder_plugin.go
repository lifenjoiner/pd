// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package http

import (
	"bufio"
	"bytes"
)

// ReqestTransformer transforms http request for direct connections.
type ReqestTransformer struct {
	Proxy bool
}

// Transform does the transformation.
func (t ReqestTransformer) Transform(b []byte) []byte {
	br := bytes.NewBuffer(b)
	r := bufio.NewReader(br)
	req, err := ParseRequest(r)
	if err != nil {
		return nil
	}

	bw := &bytes.Buffer{}
	err = req.writeRequest(bw, t.Proxy)
	if err == nil && len(req.PostData) > 0 {
		_, err = bw.Write(req.PostData)
	}
	if err != nil {
		return nil
	}
	return bw.Bytes()
}
