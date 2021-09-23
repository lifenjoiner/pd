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
const BufferSize int = 8192

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
		LeftBuf := make([]byte, BufferSize)
		for {
			fw.LeftConn.SetDeadline(time.Now().Add(LeftTimeout))
			n, LrErr = fw.LeftConn.R.Read(LeftBuf)
			if LrErr == nil {
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
	}()

	RightBuf := make([]byte, BufferSize)
	for {
		var n int
		fw.RightConn.SetDeadline(time.Now().Add(RightTimeout))
		n, RrErr = fw.RightConn.R.Read(RightBuf)
		if RrErr == nil {
			// RightBuf has enough space.
			if TlsStage == 0x00 {
				TlsStage = RightBuf[0]
				if RightBuf[0] == TlsHandshake && RightBuf[1] == 0x03 {
					// Tls v1.2, a: ServerHello + Certificate + ServerKeyExchange + ServerHelloDone
					// Tls v1.2, b: ServerHello + ChangeCipherSpec + EncryptedHandshakeMessage
					// Tls v1.3: ServerHello + ChangeCipherSpec + ApplicationData
					//log.Printf("[forwarder] TLS server Handshake data is got: %v <-- %v", fw.LeftAddr, fw.RightAddr)
				} else {
					gotRightData = true
				}
			} else if TlsStage == TlsHandshake {
				if (RightBuf[0] == TlsHandshake || RightBuf[0] == TlsChangeCipher) && RightBuf[1] == 0x03 {
					// Tls v1.2, a: [NewSessionTicket + ]ChangeCipherSpec + EncryptedHandshakeMessage
					// Weixin server sleeps (25s) before sending application data for heartbeats.
					LeftTimeout = LeftTlsAlive
					RightTimeout = RightTlsAlive
				} else if RightBuf[0] == TlsApplication && RightBuf[1] == 0x03 {
					//log.Printf("[forwarder] TLS Application data is got: %v <-- %v", fw.LeftAddr, fw.RightAddr)
					TlsStage = TlsApplication
					gotRightData = true
					LeftTimeout = LeftTlsAlive
					RightTimeout = RightTlsAlive
				} else {
					TlsStage = 0xff
					gotRightData = false
				}
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
