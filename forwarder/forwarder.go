// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// Package forwarder is the MITM that operates the communication without descryption.
package forwarder

import (
	//"log"
	"net"
	"sync"
	"time"

	"github.com/lifenjoiner/pd/bufconn"
)

// Forwarder is the relay for client to upstream and proxy to downstream.
type Forwarder struct {
	LeftAddr  net.Addr
	LeftConn  *bufconn.Conn
	LeftTran  Transformer
	RightAddr net.Addr
	RightConn *bufconn.Conn // TCP/UDP
	RightTran Transformer
	Timeout   time.Duration
	Wave      float64
}

// The reading size, could > 4k, need big enough to get the whole TLS Handshake packets.
// ServerHello + Certificate + ServerHelloDone
//
// Better Fragmentation for Loopback:
// Header length: IPv4 20 [~ 60] octets, IPv6 40 octets, TCP 20 [~ 60] octets; Options are rare.
// IP MTU = 64k (65535 bytes), so, TCP MSS = 65495 [~ 65415] bytes.
// While Ethernet v2 (WAN) MTU = 1500 bytes, yields TCP MSS = 1460 [~ 1380] bytes.
// Fragmentation: https://en.wikipedia.org/wiki/IP_fragmentation
// 500k = 512000 = 7 * 65495 + 53535 = 350 * 1460 + 1000, 1460 - 1000 = 460
// Set BufferSize big enough to reduce `cgocall` costing CPU usage, but not too big, that will consume lots of memory!
// For high/full speed traffic calling WSARecv/WSASend.
// 4k for direct `ReadFrom` is observed, and high CPU usage.
const (
	maxBufferSize int = 1024 * 1000
	minBufferSize int = 1024 * 100
)

var bufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, minBufferSize)
		return &b
	},
}

// TLS packet types.
const (
	TLSHandshake    byte = 0x16
	TLSChangeCipher byte = 0x14
	TLSApplication  byte = 0x17
)

// Tunnel operates the communication.
func (fw *Forwarder) Tunnel() (bool, error) {
	var wg sync.WaitGroup
	var LrErr, LwErr, RrErr, RwErr error

	LeftTimeout := 2 * fw.Timeout
	RightTimeout := fw.Timeout
	// Persisting is good for reuse to send without re-Handshake (SNI).
	LeftTLSAlive := time.Duration((1 + fw.Wave) * 10 * float64(fw.Timeout))
	RightTLSAlive := LeftTLSAlive + fw.Timeout
	TLSStageRight := byte(0)
	gotRightData := false

	wg.Add(1)
	go func() {
		defer wg.Done()
		var n int
		leftBufPtr := bufPool.Get().(*[]byte)
		LeftBuf := *leftBufPtr
		for {
			if x := cap(LeftBuf); n == x && x < maxBufferSize {
				LeftBuf = make([]byte, x+minBufferSize)
			}
			_ = fw.LeftConn.SetDeadline(time.Now().Add(LeftTimeout))
			n, LrErr = fw.LeftConn.R.Read(LeftBuf)
			if LrErr == nil {
				if TLSStageRight == TLSHandshake && LeftBuf[0] == TLSApplication && n > 1 && LeftBuf[1] == 0x03 {
					// Request data is sent. Some server may response slowly: snapshot downloading from https://repo.or.cz
					//log.Printf("[forwarder] TLS Application data is got: %v --> %v", fw.LeftAddr, fw.RightAddr)
					LeftTimeout = LeftTLSAlive
					RightTimeout = RightTLSAlive
				}
				//log.Printf("[forwarder] %v --> %v Read: %v", fw.LeftAddr, fw.RightAddr, n)
				data := LeftBuf[0:n]
				if fw.LeftTran != nil {
					d := fw.LeftTran.Transform(data)
					if d != nil {
						data = d
					}
				}
				_ = fw.RightConn.SetDeadline(time.Now().Add(RightTimeout))
				_, RwErr = fw.RightConn.Write(data)
			}
			if LrErr != nil || LwErr != nil || RrErr != nil || RwErr != nil {
				if isReset(LrErr) || isTimeout(LrErr) {
					_ = fw.RightConn.SetDeadline(time.Now())
				}
				_ = fw.LeftConn.SetDeadline(time.Now().Add(LeftTimeout))
				//log.Printf("[forwarder] %v --> %v: %v", fw.LeftAddr, fw.RightAddr, LrErr)
				break
			}
		}
		*leftBufPtr = LeftBuf
		bufPool.Put(leftBufPtr)
	}()

	var n int
	rightBufPtr := bufPool.Get().(*[]byte)
	RightBuf := *rightBufPtr
	for {
		if x := cap(RightBuf); n == x && x < maxBufferSize {
			RightBuf = make([]byte, x+minBufferSize)
		}
		_ = fw.RightConn.SetDeadline(time.Now().Add(RightTimeout))
		n, RrErr = fw.RightConn.R.Read(RightBuf)
		if RrErr == nil {
			// RightBuf has enough space.
			if TLSStageRight == 0x00 {
				TLSStageRight = RightBuf[0]
				if !(RightBuf[0] == TLSHandshake && n > 1 && RightBuf[1] == 0x03) {
					gotRightData = true
					//} else {
					// TLS v1.2, a: ServerHello + Certificate + ServerKeyExchange + ServerHelloDone
					// TLS v1.2, b: ServerHello + ChangeCipherSpec + EncryptedHandshakeMessage
					// TLS v1.3: ServerHello + ChangeCipherSpec + ApplicationData
					//log.Printf("[forwarder] TLS server Handshake data is got: %v <-- %v", fw.LeftAddr, fw.RightAddr)
				}
			} else if TLSStageRight == TLSHandshake {
				if (RightBuf[0] == TLSHandshake || RightBuf[0] == TLSChangeCipher) && n > 1 && RightBuf[1] == 0x03 {
					// TLS v1.2, a: [NewSessionTicket + ]ChangeCipherSpec + EncryptedHandshakeMessage
					// Weixin server sleeps (25s) before sending application data for heartbeats.
					LeftTimeout = LeftTLSAlive
					RightTimeout = RightTLSAlive
				} else if RightBuf[0] == TLSApplication && n > 1 && RightBuf[1] == 0x03 {
					// Response data is received.
					//log.Printf("[forwarder] TLS Application data is got: %v <-- %v", fw.LeftAddr, fw.RightAddr)
					TLSStageRight = TLSApplication
					gotRightData = true
					LeftTimeout = LeftTLSAlive
					RightTimeout = RightTLSAlive
				}
			}
			data := RightBuf[0:n]
			if fw.RightTran != nil {
				d := fw.RightTran.Transform(data)
				if d != nil {
					data = d
				}
			}
			_ = fw.LeftConn.SetDeadline(time.Now().Add(LeftTimeout))
			_, LwErr = fw.LeftConn.Write(data)
		}
		if LrErr != nil || LwErr != nil || RrErr != nil || RwErr != nil {
			_ = fw.LeftConn.SetDeadline(time.Now())
			//log.Printf("[forwarder] %v <-- %v: %v", fw.LeftAddr, fw.RightAddr, RrErr)
			break
		}
	}
	*rightBufPtr = RightBuf
	bufPool.Put(rightBufPtr)
	wg.Wait()

	_ = fw.RightConn.SetDeadline(time.Now())
	_ = fw.LeftConn.SetDeadline(time.Now())
	ok := gotRightData || isReset(LrErr) || isEOF(LrErr)
	//log.Print(LrErr)
	//log.Print(LwErr)
	//log.Print(RrErr)
	//log.Print(RwErr)
	//log.Print(ok)
	if ok {
		return false, nil
	}
	restart := TLSStageRight == TLSHandshake || TLSStageRight == TLSApplication
	return restart, RrErr
}
