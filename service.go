package gserv

import (
	"errors"
	"fmt"
	"github.com/raoptimus/rlog"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"
)

type (
	service struct {
		Logger   *rlog.Logger
		Start    func()
		Stop     func()
		Location *time.Location
		MaxProc  int
	}
)

var Service *service

func init() {
	Service = &service{}
}

func (s *service) Exists() bool {
	exists, err := pid.writeLock()
	if err != nil {
		log.Fatalln(err)
	}
	return exists
}

func (s *service) Run(wait bool) {
	if err := s.valid(); err != nil {
		log.Fatalln(err)
		return
	}
	log.Println("Time:", time.Now())
	log.Println("CPU:", s.MaxProc)
	log.Println("Service starting...")

	if wait {
		s.Go(s.Start)
	} else {
		s.Start()
	}
	log.Println("Service is started")

	if wait {
		s.wait()
	}
	if s.Stop != nil {
		log.Println("Service stopping...")
		s.Stop()
	}

	log.Println("Bye-Bye!")
}

func (s *service) Go(call func()) {
	defer s.DontPanic()
	call()
}

func (s *service) GetTimeMoskow() *time.Location {
	l, _ := time.LoadLocation("Etc/GMT-3")
	return l
}

func (s *service) DontPanic() {
	if r := recover(); r != nil {
		s.Logger.Crit(fmt.Sprintf("%v", r))
		log.Println(s.printStack(4))
	}
}

func (s *service) wait() {
	log.Println("Wait signals...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	log.Println("Got signal:", <-c)
}

func (s *service) valid() error {
	if s.Location == nil {
		s.Location = time.UTC
	}
	if s.Start == nil {
		return errors.New("Func 'Start' is not exists")
	}
	if s.MaxProc < 1 {
		s.MaxProc = runtime.NumCPU()
	}
	if s.Logger == nil {
		s.Logger, _ = rlog.NewLogger(rlog.LoggerTypeStd, "")
	}

	return nil
}

func (s *service) printStack(calls int) string {
	buf := make([]byte, 512)
	n := runtime.Stack(buf, false)
	if n < len(buf) {
		buf = buf[:n]
	}
	sb := string(buf)
	if calls < 0 {
		return sb
	}
	lines := 0
	i := strings.IndexFunc(sb, func(r rune) bool {
		if r == '\n' {
			lines++
			return lines > 2*calls
		}
		return false
	})
	if i > 0 {
		sb = sb[:i]
	}
	return sb
}
