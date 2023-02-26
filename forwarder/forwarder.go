// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

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
	RightAddr net.Addr
	RightConn *bufconn.Conn // TCP/UDP
	Timeout   time.Duration
}

// The reading size, could > 4k, need big enough to get the whole Tls Handshake packets.
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
		return make([]byte, minBufferSize)
	},
}

const (
	TlsHandshake    byte = 0x16
	TlsChangeCipher byte = 0x14
	TlsApplication  byte = 0x17
)

func (fw *Forwarder) Tunnel() error {
	var wg sync.WaitGroup
	var LrErr, LwErr, RrErr, RwErr error

	LeftTimeout := 2 * fw.Timeout
	RightTimeout := fw.Timeout
	// Persisting is good for reuse to send without re-Handshake (SNI).
	LeftTlsAlive := 20 * fw.Timeout
	RightTlsAlive := LeftTlsAlive + fw.Timeout
	TlsStage := byte(0)
	gotRightData := false

	wg.Add(1)
	go func() {
		defer wg.Done()
		var n int
		LeftBuf := bufPool.Get().([]byte)
		for {
			if x := cap(LeftBuf); n == x && x < maxBufferSize {
				LeftBuf = make([]byte, x+minBufferSize)
			}
			fw.LeftConn.SetDeadline(time.Now().Add(LeftTimeout))
			n, LrErr = fw.LeftConn.R.Read(LeftBuf)
			if LrErr == nil {
				if TlsStage == TlsHandshake && LeftBuf[0] == TlsApplication && n > 1 && LeftBuf[1] == 0x03 {
					// Request data is sent. Some server may response slowly: snapshot downloading from https://repo.or.cz
					//log.Printf("[forwarder] TLS Application data is got: %v --> %v", fw.LeftAddr, fw.RightAddr)
					TlsStage = TlsApplication
					LeftTimeout = LeftTlsAlive
					RightTimeout = RightTlsAlive
				}
				//log.Printf("[forwarder] %v --> %v Read: %v", fw.LeftAddr, fw.RightAddr, n)
				fw.RightConn.SetDeadline(time.Now().Add(RightTimeout))
				_, RwErr = fw.RightConn.Write(LeftBuf[0:n])
			}
			if LrErr != nil || LwErr != nil || RrErr != nil || RwErr != nil {
				if isReset(LrErr) || isTimeout(LrErr) {
					fw.RightConn.SetDeadline(time.Now())
				}
				fw.LeftConn.SetDeadline(time.Now().Add(LeftTimeout))
				//log.Printf("[forwarder] %v --> %v: %v", fw.LeftAddr, fw.RightAddr, LrErr)
				break
			}
		}
		bufPool.Put(LeftBuf)
	}()

	var n int
	RightBuf := bufPool.Get().([]byte)
	for {
		if x := cap(RightBuf); n == x && x < maxBufferSize {
			RightBuf = make([]byte, x+minBufferSize)
		}
		fw.RightConn.SetDeadline(time.Now().Add(RightTimeout))
		n, RrErr = fw.RightConn.R.Read(RightBuf)
		if RrErr == nil {
			// RightBuf has enough space.
			if TlsStage == 0x00 {
				TlsStage = RightBuf[0]
				if RightBuf[0] == TlsHandshake && n > 1 && RightBuf[1] == 0x03 {
					// Tls v1.2, a: ServerHello + Certificate + ServerKeyExchange + ServerHelloDone
					// Tls v1.2, b: ServerHello + ChangeCipherSpec + EncryptedHandshakeMessage
					// Tls v1.3: ServerHello + ChangeCipherSpec + ApplicationData
					//log.Printf("[forwarder] TLS server Handshake data is got: %v <-- %v", fw.LeftAddr, fw.RightAddr)
				} else {
					gotRightData = true
				}
			} else if TlsStage == TlsHandshake {
				if (RightBuf[0] == TlsHandshake || RightBuf[0] == TlsChangeCipher) && n > 1 && RightBuf[1] == 0x03 {
					// Tls v1.2, a: [NewSessionTicket + ]ChangeCipherSpec + EncryptedHandshakeMessage
					// Weixin server sleeps (25s) before sending application data for heartbeats.
					LeftTimeout = LeftTlsAlive
					RightTimeout = RightTlsAlive
				} else if RightBuf[0] == TlsApplication && n > 1 && RightBuf[1] == 0x03 {
					// Response data is received.
					//log.Printf("[forwarder] TLS Application data is got: %v <-- %v", fw.LeftAddr, fw.RightAddr)
					TlsStage = TlsApplication
					gotRightData = true
					LeftTimeout = LeftTlsAlive
					RightTimeout = RightTlsAlive
				} else {
					// Certificate fragmentation has been observed.
				}
			} else if TlsStage == TlsApplication {
				gotRightData = true
			}
			fw.LeftConn.SetDeadline(time.Now().Add(LeftTimeout))
			_, LwErr = fw.LeftConn.Write(RightBuf[0:n])
		}
		if LrErr != nil || LwErr != nil || RrErr != nil || RwErr != nil {
			fw.LeftConn.SetDeadline(time.Now())
			//log.Printf("[forwarder] %v <-- %v: %v", fw.LeftAddr, fw.RightAddr, RrErr)
			break
		}
	}
	bufPool.Put(RightBuf)
	wg.Wait()

	fw.RightConn.SetDeadline(time.Now())
	fw.RightConn.Close()
	ok := gotRightData || isReset(LrErr) || isEOF(LrErr)
	//log.Print(LrErr)
	//log.Print(LwErr)
	//log.Print(RrErr)
	//log.Print(RwErr)
	//log.Print(ok)
	if ok {
		fw.LeftConn.SetDeadline(time.Now())
		fw.LeftConn.Close()
		return nil
	} else {
		fw.LeftConn.SetDeadline(time.Now().Add(LeftTimeout))
		return RrErr
	}
}
