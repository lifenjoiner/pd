// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package http

import (
	"bufio"
	"errors"
	"io"
	"net/textproto"
	"net/url"
	"time"

	"github.com/lifenjoiner/pd/bufconn"
	"github.com/lifenjoiner/pd/forwarder"
	"github.com/lifenjoiner/pd/protocol"
)

type Request struct {
	Method    string
	url       string
	Proto     string
	Auth      string
	PostData  []byte // to retry to defend RST
	TlsData   []byte // to retry to defend RST, ClientHello
	Responsed bool

	Header   textproto.MIMEHeader
	URL      *url.URL
	TryCount byte
}

func (r *Request) Command() string {
	return r.Method
}

func (r *Request) Target() string {
	return r.url
}

func (r *Request) Host() string {
	return r.URL.Host
}

func (r *Request) Hostname() string {
	return r.URL.Hostname()
}

func (r *Request) Port() string {
	return protocol.GetPort(r.URL)
}

// Request the ClientHello for sending to a remote server.
// RCWN (Race Cache With Network) or ads blockers would abort dial-in without sendig ClientHello! Drop it.
func (r *Request) GetRequest(w io.Writer, rd *bufio.Reader) (err error) {
	if !r.Responsed {
		_, err = w.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		r.Responsed = true
		if err == nil {
			err = r.cacheTlsData(rd)
		}
	}
	return
}

func (r *Request) Request(fw *forwarder.Forwarder, seg bool) (restart bool, err error) {
	_ = fw.LeftConn.SetDeadline(time.Now().Add(2 * fw.Timeout))
	_ = fw.RightConn.SetDeadline(time.Now().Add(fw.Timeout))
	if r.Method == "CONNECT" {
		if len(r.TlsData) > 0 {
			if seg {
				_, err = fw.RightConn.SplitWrite(r.TlsData, 6)
			} else {
				_, err = fw.RightConn.Write(r.TlsData)
			}
		} else {
			// drop it
			return false, nil
		}
	} else {
		err = r.writeRequest(fw.RightConn)
		if err == nil && len(r.PostData) > 0 {
			_, err = fw.RightConn.Write(r.PostData)
		}
	}
	if err == nil {
		restart, err = fw.Tunnel()
	}
	return
}

func (r *Request) cacheTlsData(rd *bufio.Reader) (err error) {
	r.TlsData, err = bufconn.ReceiveData(rd)
	return
}

func (r *Request) writeRequest(w io.Writer) (err error) {
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
	err = writeStartLine(w, r.Method, r.URL.RequestURI(), r.Proto)

	if err != nil {
		return
	}
	err = writeHeaders(w, r.Header)
	return
}

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

	r.PostData, err = bufconn.ReadData(rd)

	return
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
