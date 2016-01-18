package gserv

import (
	"testing"
	"time"
	"github.com/raoptimus/gserv/config"
)

func TestServeOne(t *testing.T) {
	started := false
	stopped := false
	Service.Start = func() {
		if config.String("config_string", "") != "testvalue" {
			t.Fatal("Config is not read")
		}
		started = true
	}
	Service.Stop = func() {
		stopped = true
	}
	Service.Exists()
	Service.Run(true)
	time.Sleep(2 * time.Second)

	if !started || !stopped {
		t.Fail()
	}
}
