package gserv
import (
	"os"
	"flag"
	"log"
	"path"
	"strconv"
	"syscall"
	"os/user"
	"errors"
	"fmt"
)

type (
	pidFs struct  {
	 fileName    string
	 file      *os.File
	}
)

var pid = &pidFs{}

func init() {
	flag.StringVar(&pid.fileName, "pid", "", "Pid file")
}

func (s *pidFs) writeLock() (exists bool, err error) {
	if !flag.Parsed() {
		flag.Parse()
	}
	if s.fileName == "" {
		s.fileName = s.getDefaultPidFile()
	}
	if s.fileName == "" {
		return false, errors.New("Pid file can't be blank")
	}
	if !s.fileExist(s.fileName) {
		if err := s.createDir(path.Dir(s.fileName)); err != nil {
			return false, fmt.Errorf("Can't create the dir %s: %v\n", s.fileName, err)
		}
	}
	s.file, err = os.OpenFile(s.fileName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return false, fmt.Errorf("Can't open pid file: %v", err)
	}
	b := make([]byte, 1024)
	if n, _ := s.file.Read(b); n > 0 {
		pid := string(b[:n])
		log.Println("Last pid:", pid)
		lastPid, err := strconv.Atoi(pid)
		if err == nil {
			p, err := os.FindProcess(lastPid)
			if err == nil {
				if err := p.Signal(syscall.Signal(0)); err == nil {
					s.closePidFs()
					return true, fmt.Errorf("The pid %s already using", pid)
				}
			}
		} else {
			log.Printf("Warninig: Last pid is not integer\n")
		}
	}
	if err := syscall.Flock(int(s.file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		s.closePidFs()
		return true, errors.New("Service already running. Pid file is locked")
	}
	if err := s.file.Truncate(0); err != nil {
		s.closePidFs()
		return false, errors.New("Can't truncate pid file")
	}
	s.file.Seek(0, 0)
	pid := os.Getpid()
	if _, err := s.file.WriteString(strconv.Itoa(pid)); err != nil {
		s.closePidFs()
		return false, errors.New("Can't be write to pid file")
	}
	log.Printf("Wrote pid %v to file %v\n", pid, s.fileName)
	return false, nil
}


func (s *pidFs) getDefaultPidFile() string {
	usr, err := user.Current()
	if err != nil {
		log.Println(err)
		return ""
	}
	exe := path.Base(os.Args[0])
	return usr.HomeDir + "/run/" + exe + ".pid"
}

func (s *pidFs) createDir(dir string) error {
	if s.fileExist(dir) {
		return nil
	}
	if err := os.Mkdir(dir, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func (s *pidFs) fileExist(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}
	return true
}

func (s *pidFs) closePidFs() {
	if s.file != nil {
		s.file.Close()
		s.file = nil
	}
}