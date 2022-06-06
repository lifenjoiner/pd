// Copyright 2021-now by lifenjoiner. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package statichost

import (
	"log"
	"testing"
)

func TestGetHostStrategy(t *testing.T) {
	log.Printf("StaticNil: %v", StaticNil)
	log.Printf("StaticDirect: %v", StaticDirect)
	log.Printf("StaticBlocked: %v", StaticBlocked)

	list := "golang.org\ncn.github.com	 # cn"

	sd := StaticHosts{}
	sd.Upsert(list, StaticDirect)
	sd.Upsert("1.1.*", StaticBlocked)
	log.Print(sd)

	var n Strategy

	n = sd.GetStrategy("github.com")
	log.Printf("github.com: %v", n)
	if n != StaticNil {
		t.Fail()
	}

	n = sd.GetStrategy("play.golang.org")
	log.Printf("play.golang.org: %v", n)
	if n != StaticDirect {
		t.Fail()
	}

	n = sd.GetStrategy("1.1.0.1")
	log.Printf("1.1.0.1: %v", n)
	if n != StaticBlocked {
		t.Fail()
	}
	n = sd.GetStrategy("127.0.0.1")
	log.Printf("127.0.0.1: %v", n)
	if n != StaticNil {
		t.Fail()
	}
}
