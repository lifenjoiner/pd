// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// The CLI.
package main

import (
	"log"
	"sync"

	"github.com/lifenjoiner/pd/dispatcher"
	"github.com/lifenjoiner/pd/hoststat"
	"github.com/lifenjoiner/pd/proxypool"
	"github.com/lifenjoiner/pd/server/tcp"
	"github.com/lifenjoiner/pd/statichost"
)

var (
	name        = "pd"
	version     = "0.1.13"
	description = "A local proxy dispatcher."
)

// ServeFromConfig starts the serving.
func ServeFromConfig(config *Config) {
	svrConf := &config.SvrConf
	dispatcher.GlobalStaticHosts = statichost.MapStaticFiles(config.Blocked, config.Direct)
	dispatcher.GlobalHostStats = hoststat.MapStatsFile(config.StatFile, config.StatValidity)
	dispatcher.StartProbeDirect(config.NetProbeURL, svrConf.UpstreamTimeout)
	go func() {
		dispatcher.GlobalProxyPool = proxypool.InitProxyPool(svrConf.Proxies, svrConf.ProxyProbeURL, svrConf.UpstreamTimeout)
	}()

	var wg sync.WaitGroup
	for _, listen := range config.Listens {
		wg.Add(1)
		s := &tcp.Server{WG: &wg, Addr: listen, Config: svrConf}
		go s.ListenAndServe()
	}
	wg.Wait()
}

func main() {
	cfg := parseConfig()
	log.Printf("%v v%v - %v", name, version, description)
	ServeFromConfig(cfg)
}
