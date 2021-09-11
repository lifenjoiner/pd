// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package http

import (
	"io"
	"net/textproto"
	"strings"
)

// parseStartLine parses "GET /foo HTTP/1.1" or "HTTP/1.1 200 OK" into its three parts.
func parseStartLine(line string) (r1, r2, r3 string, ok bool) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return
	}
	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], true
}

func writeStartLine(w io.Writer, s1, s2, s3 string) (err error) {
	_, err = io.WriteString(w, s1 +" "+ s2 +" "+ s3 +"\r\n")
	return
}

func writeHeaders(w io.Writer, header textproto.MIMEHeader) (err error) {
	for key, values := range header {
		for _, v := range values {
			_, err = io.WriteString(w, key +": "+ v +"\r\n")
			if err != nil {
				return
			}
		}
	}
	_, err = io.WriteString(w, "\r\n")
	return
}
