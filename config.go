// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"flag"
	"strings"
	"time"

	"github.com/lifenjoiner/pd/server"
)

type Config struct {
	Listens     []string
	NetProbeURL string
	SvrConf     server.Config
	StatFile    string
	Blocked     string
	Direct      string
}

func parseConfig() *Config {
	conf := &Config{}

	s := flag.String("listens", "127.0.0.1:6699", "Listen addresses: [Host]:Port[,[Host]:Port][...]")
	flag.DurationVar(&conf.SvrConf.UpstreamTimeout, "upstreamtimeout", 5*time.Second, "LookupHost/Dial/HandShake timeout, 3-7s is recommended. 20 * me for data transfer.")
	flag.StringVar(&conf.NetProbeURL, "netprobeurl", "https://example.com", "Used to probe if we are offline, and to ignore offline failures.")
	flag.BoolVar(&conf.SvrConf.ParallelDial, "paralleldial", true, "Try parallelly dial up IPs of a host.")
	flag.StringVar(&conf.SvrConf.Proxies, "proxies", "", "Upstream proxy urls: [Scheme://]Host:Port[,[Scheme://]Host:Port][...], omitting scheme adopts all supported schemes (http, socks5, socks4a).")
	flag.StringVar(&conf.SvrConf.ProxyProbeUrl, "proxyprobeurl", "https://www.google.com", "Used to probe if a proxy works.")
	flag.StringVar(&conf.StatFile, "statfile", "stat.json", "File of periodic (24h) direct connection quality (EWMA of the last 10).")
	flag.StringVar(&conf.Blocked, "blocked", "blocked", "File of blocked domains (suffix) or IPs (prefix), that go proxied directly. Do 1 direct try, if no proxy.")
	flag.StringVar(&conf.Direct, "direct", "direct", "File of direct domains (suffix) or IPs (prefix), that won't go proxied. Direct > Blocked.")

	flag.Parse()
	conf.Listens = strings.Split(*s, ",")

	return conf
}
