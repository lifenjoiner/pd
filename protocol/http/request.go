// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package http

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/protocol"
)

// Request struct.
type Request struct {
	Method    string
	url       string
	Proto     string
	Auth      string
	PostData  []byte // to retry to defend RST
	TLSData   []byte // to retry to defend RST, ClientHello
	Responsed bool

	Header   textproto.MIMEHeader
	URL      *url.URL
	TryCount byte
}

// Command requested by.
func (r *Request) Command() string {
	return r.Method
}

// Target URL requested to.
func (r *Request) Target() string {
	return r.url
}

// Host requested to.
func (r *Request) Host() string {
	return r.URL.Host
}

// Hostname only requested to.
func (r *Request) Hostname() string {
	return r.URL.Hostname()
}

// Port requested to.
func (r *Request) Port() string {
	return protocol.GetPort(r.URL)
}

// GetInnerRequest requests the ClientHello for sending to a remote server.
// RCWN (Race Cache With Network) or ads blockers would abort dial-in without sendig ClientHello! Drop it.
func (r *Request) GetInnerRequest(c *bufconn.Conn) (err error) {
	if !r.Responsed {
		_, err = c.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		r.Responsed = true
		if err == nil {
			r.TLSData, err = c.ReadAll()
		}
	}
	return
}

// Request to a upstream server.
func (r *Request) Request(c *bufconn.Conn, proxy, seg bool) (err error) {
	if r.Method == "CONNECT" {
		if len(r.TLSData) > 0 {
			if seg {
				h := []byte(r.URL.Hostname())
				i := bytes.Index(r.TLSData, h)
				i += len(h) / 2
				_, err = c.SplitWrite(r.TLSData, i)
			} else {
				_, err = c.Write(r.TLSData)
			}
		} else {
			// drop it
			return nil
		}
	} else {
		if seg {
			err = r.writeRequest(c, proxy)
		} else {
			bw := &bytes.Buffer{}
			err = r.writeRequest(bw, proxy)
			if err == nil {
				_, err = c.Write(bw.Bytes())
			}
		}
		if err == nil && len(r.PostData) > 0 {
			_, err = c.Write(r.PostData)
		}
	}
	return
}

func (r *Request) writeRequest(w io.Writer, proxy bool) (err error) {
	// NTLMSSP automatic logon requires `Keep-Alive`.
	nc := false
	cv := r.Header.Get("Connection")
	if cv == "" {
		// Windows set `Proxy-Connection` rather than `Connection`.
		nc = true
		cv = r.Header.Get("Proxy-Connection")
	}
	if cv == "" {
		cv = "close"
	}

	cleanHeaders(r.Header)

	if nc {
		r.Header.Set("Connection", cv)
	}
	// Proxy Authorization: LAN proxy doesn't need, in WAN it is BLOCKED!
	r.Header.Set("Host", r.URL.Host)

	path := r.URL.RequestURI()
	if proxy {
		path = r.url
	}
	err = writeStartLine(w, r.Method, path, r.Proto)
	if err != nil {
		return
	}
	err = writeHeaders(w, r.Header)
	return
}

// ParseRequest parses a request.
func ParseRequest(rd *bufio.Reader) (r *Request, err error) {
	tpr := textproto.NewReader(rd)
	line, err := tpr.ReadLine()
	if err != nil {
		return nil, err
	}

	var ok bool
	r = &Request{}
	r.Method, r.url, r.Proto, ok = parseStartLine(line)
	if !ok {
		return nil, errors.New("malformed HTTP start line")
	}

	r.Header, err = tpr.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}

	// The net/rpc package also uses CONNECT.
	rawURL := r.url
	if r.Method == "CONNECT" {
		rawURL = "//" + rawURL
	}
	r.URL, err = url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	r.PostData, err = bufconn.ReadAll(rd)

	return
}

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

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Connection
// https://en.wikipedia.org/wiki/List_of_HTTP_header_fields
// Reuse "Connection", "Keep-Alive" and "Upgrade" (websocket).
func cleanHeaders(header textproto.MIMEHeader) {
	hopByHopHeaders := []string{
		"Proxy-Connection", // Implemented as a misunderstanding of the HTTP specifications
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailer",
		"Transfer-Encoding",
	}
	for _, h := range hopByHopHeaders {
		header.Del(h)
	}
}

func writeStartLine(w io.Writer, s1, s2, s3 string) (err error) {
	_, err = io.WriteString(w, s1+" "+s2+" "+s3+"\r\n")
	return
}

func writeHeaders(w io.Writer, header textproto.MIMEHeader) (err error) {
	for key, values := range header {
		for _, v := range values {
			_, err = io.WriteString(w, key+": "+v+"\r\n")
			if err != nil {
				return
			}
		}
	}
	_, err = io.WriteString(w, "\r\n")
	return
}
