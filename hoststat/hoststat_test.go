// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package hoststat

import (
	"log"
	"math"
	"math/rand"
	"os"
	"time"
	"testing"
)

func TestHostStats(t *testing.T) {
	var hs *HostStats
	file := os.TempDir() +"/stat.json"
	h := "github.com:https"
	//
	os.Remove(file)
	hs = &HostStats{}
	hs.Load(file)
	rand.Seed(time.Now().UnixNano())
	N := 10
	for i := 1; i <= N; i++ {
		hs.Update(h, 1)
		v := hs.Stats[h].Value
		log.Printf("%v: %v", i, v)
		if v != 1 || hs.Stats[h].Count != i {
			t.Fail()
		}
	}
	if hs.Stats[h].Count != N {
		t.Fail()
	}
	hs = &HostStats{Stats: make(map[string]*HostStat)}
	for i := 1; i <= N; i++ {
		hs.Update(h, 0)
		v := hs.Stats[h].Value
		log.Printf("%v: %v", i, v)
		if v != 0 {
			t.Fail()
		}
	}
	Total := 0.0
	for i := 1; i <= 2 * N; i++ {
		v := float64(rand.Intn(2))
		hs.Update(h, v)
		log.Printf("%v: %v -> %v", i, v, hs.Stats[h].Value)
		if i > N {
			Total += v
		}
	}
	delta := Total / float64(N) - hs.Stats[h].Value
	log.Printf("Average - Value = %v", delta)
	if math.Abs(delta) > 1 / float64(N) {
		t.Fail()
	}
	hs.Save(file)
	hs.Validity = time.Millisecond
	time.Sleep(2 * time.Millisecond)
	hs.Cleanup()
	if len(hs.Stats) != 0 {
		t.Fail()
	}
}