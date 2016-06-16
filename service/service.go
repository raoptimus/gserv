package service

import (
	"errors"
	"flag"
	"fmt"
	"github.com/raoptimus/rlog"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

type (
	service struct {
		*BaseService
	}
	BaseService struct {
		Logger   *rlog.Logger
		Start    func()
		Stop     func()
		Location *time.Location
		MaxProc  int
	}
)

var serv = &service{}

func Init(bs *BaseService) {
	serv.BaseService = bs
}

func Exists() bool {
	exists, err := pid.writeLock()
	if err != nil {
		log.Fatalln(err)
	}
	return exists
}

func Start(wait bool) {
	if err := serv.valid(); err != nil {
		log.Fatalln(err)
		return
	}
	time.Local = serv.Location
	log.Println("Time:", time.Now())
	log.Println("CPU:", serv.MaxProc)
	log.Println("Service starting...")

	if wait {
		Go(serv.Start)
	} else {
		serv.Start()
	}
	log.Println("Service is started")

	if wait {
		serv.wait()
	}
	if serv.Stop != nil {
		log.Println("Service stopping...")
		serv.Stop()
	}

	log.Println("Bye-Bye!")
}

func Go(call func()) {
	go func(call func()) {
		defer DontPanic()
		call()
	}(call)
}

func GetTimeMoskow() *time.Location {
	l, _ := time.LoadLocation("Etc/GMT-3")
	return l
}

func DontPanic() {
	if r := recover(); r != nil {
		serv.Logger.Crit(fmt.Sprintf("%v", r))
		log.Println(serv.printStack(4))
	}
}

func StartProfiler(addr string) {
	go func(addr string) {
		defer DontPanic()
		var netprofile = flag.Bool(
			"netprofile",
			true,
			"record profile; see http://"+addr+"/debug/prof",
		)
		var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
		var memprofile = flag.String("memprofile", "", "write memory profile to this file")
		flag.Parse()

		if *netprofile {
			go func() {
				log.Printf("Profiling enabled and bind addr: %s\n", addr)
				log.Printf("See profiling in http://%s/debug/pprof/\n", addr)
				log.Printf("See log requests in curl -s http://%s/debug/vars | json queries | json -a\n", addr)

				if err := http.ListenAndServe(addr, nil); err != nil {
					log.Println(err)
					return
				}
			}()
		}

		if *cpuprofile != "" {
			f, err := os.Create(*cpuprofile)
			if err != nil {
				log.Println(err)
				return
			}

			pprof.StartCPUProfile(f)
		}

		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				fmt.Println(err)
				return
			}

			pprof.WriteHeapProfile(f)
			f.Close()
			return
		}
	}(addr)
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
