package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type (
	config struct {
		sync.RWMutex
		data  *configData
		after *after
	}
	after struct {
		sync.RWMutex
		events map[string]func()
	}
	configData map[string]json.RawMessage
)

var cfg *config

func OnAfterLoad(name string, f func()) {
	cfg.after.Lock()
	defer cfg.after.Unlock()
	cfg.after.events[name] = f
}

func OffAfterLoad(name string) {
	cfg.after.RLock()
	_, ok := cfg.after.events[name]
	cfg.after.RUnlock()
	if !ok {
		return
	}
	cfg.after.Lock()
	defer cfg.after.Unlock()
	delete(cfg.after.events, name)
}

func Object(name string, value interface{}) error {
	return cfg.get(name, &value)
}

func String(name, value string) string {
	var v string
	if err := cfg.get(name, &v); err != nil {
		return value
	}
	return v
}

func Bool(name string, value bool) bool {
	var v bool
	if err := cfg.get(name, &v); err != nil {
		return value
	}
	return v
}

func Int(name string, value int) int {
	var v int
	if err := cfg.get(name, &v); err != nil {
		return value
	}
	return v
}

func Duration(name string, value time.Duration) time.Duration {
	var v string
	if err := cfg.get(name, &v); err != nil {
		return value
	}
	dur, err := time.ParseDuration(v)
	if err == nil {
		return dur
	}
	log.Printf("Config: cannot parse duration of %q: %v", name, err)
	return value
}

func (s *config) get(name string, value interface{}) error {
	s.RLock()
	defer s.RUnlock()

	data, ok := (*s.data)[name]
	if !ok {
		err := errors.New(fmt.Sprintf("Not found the key %s in config", name))
		log.Println(err)
		return err
	}
	if err := json.Unmarshal(data, &value); err != nil {
		err = errors.New(fmt.Sprintf("Getting config value of '%s': %v", name, err))
		log.Println(err)
		return err
	}
	return nil
}

func loadConfig(cfgFileName string) (*configData, error) {
	if cfgFileName == "" {
		log.Fatalln("Config file name is not defined. Add -config=path argument")
	}
	file, err := ioutil.ReadFile(cfgFileName)
	if err != nil {
		return nil, fmt.Errorf("Can't opened config file %s: %v\n", cfgFileName, err)
	}

	var d configData
	if err = json.Unmarshal(file, &d); err != nil {
		return nil, fmt.Errorf("Can't parsed config file %s: %v\n", cfgFileName, err)
	}

	log.Printf("The config file %s is loaded", cfgFileName)
	return &d, nil
}

func Init(configFile string) {
	if cfg != nil {
		return
	}
	d, err := loadConfig(configFile)
	if err != nil {
		log.Fatalln(err)
	}
	cfg = &config{
		data: d,
		after: &after{
			events: make(map[string]func()),
		},
	}
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGUSR2)
	go func() {
		for {
			<-s
			d, err = loadConfig(configFile)
			if err != nil {
				log.Printf("Can't loaded the config file %s: %v\n", configFile, err)
				return
			}
			cfg.Lock()
			cfg.data = d
			cfg.Unlock()

			go cfg.raise()
		}
	}()
}

func (s *config) raise() {
	s.after.RLock()
	defer s.after.RUnlock()

	for _, f := range s.after.events {
		f()
	}
}
