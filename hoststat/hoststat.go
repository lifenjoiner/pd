// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package hoststat

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/lifenjoiner/ewma"
)

/* HostStats example:
{
	"github.com:443": {
		"v": 0.7,
		"n": 10,
		"t": "2021-08-18T21:46:05.9266165+08:00"
	}
}
*/

// The EWMA window size.
const EwmaSlide int = 10

// A single HostStat.
type HostStat struct {
	Value float64   `json:"v"`
	Count int       `json:"n"`
	Time  time.Time `json:"t"`
	ewma  *ewma.EWMA
}

// The whole HostStats.
type HostStats struct {
	sync.RWMutex
	Stats          map[string]*HostStat
	Validity       time.Duration
	BackupInterval time.Duration
	LastRecount    time.Time
}

// Get the HostStat.
func (hs *HostStats) GetStat(h string) (stat HostStat) {
	hs.RLock()
	st := hs.Stats[h]
	if st != nil {
		stat = *st
	}
	hs.RUnlock()
	return
}

// Update a host's stat by new value.
func (hs *HostStats) Update(h string, v float64) {
	hs.Lock()
	stats := hs.Stats
	if stats[h] == nil {
		//stats.Stats = make(map[string]*HostStat)
		stats[h] = &HostStat{}
	}
	stat := stats[h]
	if hs.Validity > 0 && time.Since(stat.Time) > hs.Validity {
		stat.Count = 0
		stat.Value = 0
	}
	if stat.ewma == nil {
		stat.ewma = ewma.NewMovingAverage(EwmaSlide)
		if stat.Count > EwmaSlide {
			stat.ewma.Set(stat.Value)
		}
	}
	stat.ewma.Add(v)
	stat.Count++
	stat.Value = stat.ewma.Value()
	stat.Time = time.Now()
	hs.Unlock()
}

// Clean expired stats up, and reset the stat periodically.
func (hs *HostStats) Cleanup() {
	if hs.Stats == nil {
		return
	}
	hs.Lock()
	stats := hs.Stats
	var changed bool
	newStats := make(map[string]*HostStat)
	for h, stat := range stats {
		if hs.Validity > 0 && time.Since(stat.Time) > hs.Validity {
			changed = true
		} else {
			if hs.Validity > 0 && time.Since(hs.LastRecount) > hs.Validity && stat.Count > EwmaSlide {
				stat.Count = EwmaSlide + 1
				changed = true
			}
			newStats[h] = stat
		}
	}
	if changed {
		hs.Stats = newStats
	}
	hs.Unlock()
}

// Load HostStats from a file.
func (hs *HostStats) Load(file string) {
	data, err := os.ReadFile(file)
	if err != nil {
		log.Printf("[hoststats] %v", err)
	}
	if len(data) == 0 {
		data = []byte("{}")
	}

	hs.Lock()
	err = json.Unmarshal(data, &hs.Stats)
	if err != nil {
		log.Printf("[hoststats] %v", err)
	}
	hs.LastRecount = time.Now()
	hs.Unlock()
	hs.Cleanup()
}

// Save HostStats to a file.
func (hs *HostStats) Save(file string) {
	hs.Cleanup()
	hs.RLock()
	data, err := json.MarshalIndent(hs.Stats, "", "\t")
	hs.RUnlock()
	if err != nil {
		log.Printf("[hoststats] %v", err)
		data = []byte("{}")
	}

	err = os.WriteFile(file, []byte(data), 0666)
	if err != nil {
		log.Printf("[hoststats] %v", err)
	}
}

// Load HostStats from a file, and save back the new results periodically.
func MapStatsFile(file string, validity time.Duration) (hs *HostStats) {
	hs = &HostStats{
		Validity:       validity,
		BackupInterval: 5 * time.Minute,
	}
	hs.Load(file)
	if hs.BackupInterval > 0 {
		go func() {
			for {
				time.Sleep(hs.BackupInterval)
				log.Printf("[hoststats] Saving: %v", file)
				hs.Save(file)
			}
		}()
	}
	return
}
