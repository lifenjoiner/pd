// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package hoststat

import (
	"log"
	"math"
	"os"
	"testing"
	"time"
)

func TestHostStats(t *testing.T) {
	var hs *HostStats
	file := os.TempDir() + "/stat.json"
	h := "github.com:https"
	//
	hs = &HostStats{}
	os.Remove(file)
	hs.Load(file)
	//
	N := 10
	for i := 1; i <= N; i++ {
		hs.Update(h, 1)
		v := hs.GetStat(h).Value
		log.Printf("%v: %v", i, v)
		if v != 1 || hs.GetStat(h).Count != i {
			t.Fail()
		}
	}
	if hs.GetStat(h).Count != N {
		t.Fail()
	}
	//
	hs = &HostStats{Stats: make(map[string]*HostStat)}
	for i := 1; i <= N; i++ {
		hs.Update(h, 0)
		v := hs.GetStat(h).Value
		log.Printf("%v: %v", i, v)
		if v != 0 {
			t.Fail()
		}
	}
	//
	hs = &HostStats{Stats: make(map[string]*HostStat)}
	var tests = [20]float64{1, 0, 0, 1, 0, 1, 1, 1, 1, 0, 0, 0, 0, 1, 0, 1, 1, 1, 1, 0}
	var ewmas = [20]float64{1.00, 0.33, 0.17, 0.50, 0.33, 0.52, 0.64, 0.72, 0.78, 0.64, 0.52, 0.43, 0.35, 0.47, 0.38, 0.49, 0.59, 0.66, 0.72, 0.59}
	for i := 0; i < 20; i++ {
		hs.Update(h, tests[i])
		st := hs.GetStat(h)
		log.Printf("%v: %v -> %v", i+1, tests[i], st.Value)
		if math.Round(st.Value*100)/100 != ewmas[i] {
			t.Errorf("expected = %v", ewmas[i])
		}
	}
	//
	hs.Save(file)
	//
	hs = &HostStats{}
	hs.Load(file)
	st := hs.GetStat(h)
	if math.Round(st.Value*100)/100 != ewmas[19] {
		t.Errorf("value = %v, expected = %v", st.Value, ewmas[19])
	}
	// GetStat
	if hs.GetStat("blog.github.com:https").Count != 0 {
		t.Fail()
	}
	//
	hs.Validity = time.Millisecond
	time.Sleep(2 * time.Millisecond)
	hs.Cleanup()
	if len(hs.Stats) != 0 {
		t.Fail()
	}
}
