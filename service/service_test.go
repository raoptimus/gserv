package service

import (
	"github.com/raoptimus/gserv/config"
	"testing"
	"time"
)

func TestServeOne(t *testing.T) {
	started := false
	stopped := false
	Init(&BaseService{
		Start: func() {
			if config.String("config_string", "") != "testvalue" {
				t.Fatal("Config is not read")
			}
			started = true
		},
		Stop: func() {
			stopped = true
		},
	})
	service.Start(false)

	if !Exists() {
		t.Fail()
	}

	time.Sleep(1 * time.Second)

	if !started || !stopped {
		t.Fail()
	}
}
