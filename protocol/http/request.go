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
	Url       string
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
	return r.Url
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

func (r *Request) Request(fw *forwarder.Forwarder, seg bool) (err error) {
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
			return nil
		}
	} else {
		err = r.writeRequest(fw.RightConn)
		if err == nil && len(r.PostData) > 0 {
			_, err = fw.RightConn.Write(r.PostData)
		}
	}
	if err == nil {
		err = fw.Tunnel()
	}
	return
}

func (r *Request) cacheTlsData(rd *bufio.Reader) (err error) {
	r.TlsData, err = bufconn.ReceiveData(rd)
	return
}

func (r *Request) writeRequest(w io.Writer) (err error) {
	var r2 *Request = &Request{}
	*r2 = *r
	cleanHeaders(r2.Header)
	// Does not support http keep-alive.
	r2.Header.Set("Connection", "close")
	// Proxy Authorization: LAN proxy doesn't need, in WAN it is BLOCKED!
	r2.Header.Set("Host", r.URL.Host)
	err = writeStartLine(w, r2.Method, r2.URL.RequestURI(), r2.Proto)
	if err != nil {
		return
	}
	err = writeHeaders(w, r2.Header)
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
	r.Method, r.Url, r.Proto, ok = parseStartLine(line)
	if !ok {
		return nil, errors.New("malformed HTTP start line")
	}

	r.Header, err = tpr.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}

	// The net/rpc package also uses CONNECT.
	rawUrl := r.Url
	if r.Method == "CONNECT" {
		rawUrl = "//" + rawUrl
	}
	r.URL, err = url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	r.PostData, err = bufconn.ReadData(rd)

	return
}

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
// https://en.wikipedia.org/wiki/List_of_HTTP_header_fields
func cleanHeaders(header textproto.MIMEHeader) {
	header.Del("Proxy-Connection")
	header.Del("Proxy-Authenticate")
	header.Del("Proxy-Authorization")
}
