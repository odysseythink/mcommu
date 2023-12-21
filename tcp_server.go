package mcommu

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"mlib.com/mrun"
)

type TCPServer struct {
	wg         sync.WaitGroup
	addr       string
	tcpConnMgr mrun.ModuleMgr
	processor  IProcessor
	maxConnNum int
	ln         net.Listener
	threadnum  int
}

func (s *TCPServer) This() ICommunicator {
	return s
}

func (s *TCPServer) Init(addr string, processor IProcessor, args ...interface{}) error {
	if addr == "" || processor == nil {
		log.Printf("[E]inavlid arg\n")
		return fmt.Errorf("[E]inavlid arg")
	}
	if !IsNetAddrValid(addr) {
		log.Printf("[E]inavlid addr(%s)\n", addr)
		return fmt.Errorf("[E]inavlid addr(%s)", addr)
	}
	s.addr = addr
	if len(args) > 0 {
		if threadnum, ok := args[0].(int); !ok || threadnum <= 0 {
			log.Printf("[E]args[0](%#v) must be a valid integer to define threadnum\n", args[0])
			return fmt.Errorf("[E]args[0](%#v) must be a valid integer to define threadnum", args[0])
		} else {
			s.threadnum = threadnum
		}
		if len(args) > 1 {
			if maxConnNum, ok := args[1].(int); !ok || maxConnNum <= 0 {
				log.Printf("[E]args[1](%#v) must be a valid integer to define maxConnNum\n", args[1])
				return fmt.Errorf("[E]args[1](%#v) must be a valid integer to define maxConnNum", args[1])
			} else {
				s.maxConnNum = maxConnNum
			}
		} else {
			s.maxConnNum = 1024
		}
	} else {
		s.threadnum = 50
		s.maxConnNum = 1024
	}
	s.processor = processor
	var err error
	err = s.tcpConnMgr.Init()
	if err != nil {
		log.Printf("[E]tcpConnMgr init failed:%v\n", err)
		return fmt.Errorf("tcpConnMgr init failed:%v", err)
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Printf("[E]net.Listen(%s) failed:%v\n", s.addr, err)
		return fmt.Errorf("net.Listen(%s) failed:%v", s.addr, err)
	}
	s.ln = ln

	s.wg.Add(1)
	go func() {
		s.run()
		s.wg.Done()
	}()
	return nil
}

func (s *TCPServer) Protocol() string {
	return "tcpserver"
}

func (s *TCPServer) run() {
	var tempDelay time.Duration

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if _, ok := err.(net.Error); ok {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Printf("[W]accept error: %v; retrying in %v\n", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return
		}
		tempDelay = 0

		connNum := s.tcpConnMgr.ModuleNum()
		if connNum >= s.maxConnNum {
			conn.Close()
			log.Printf("[W]too many connections\n")
			continue
		}
		log.Printf("[D]remote(%s) connected\n", conn.RemoteAddr().String())
		tcpconn := &tcpConn{}
		err = s.tcpConnMgr.Register(tcpconn, []mrun.ModuleMgrOption{mrun.NewModuleErrorOption(s.onError)}, conn, s.processor)
		if err != nil {
			log.Printf("[W]conn retister failed:%v\n", err)
			conn.Close()
			continue
		}
	}
}

func (s *TCPServer) Close() {
	if s.ln != nil {
		s.ln.Close()
	}
	s.tcpConnMgr.Destroy()
	s.wg.Wait()
}

func (server *TCPServer) SendToRemote(addr string, msg interface{}) error {
	var err error
	exist := false
	server.tcpConnMgr.Range(func(m mrun.IModule) bool {
		if m != nil && m.UserData() != nil {
			if conn, ok := m.UserData().(IConn); ok && conn != nil {
				if conn.RemoteAddr() == addr {
					err = conn.Write(msg)
					exist = true
					return false
				}
			}
		}
		return true
	})
	if !exist {
		return fmt.Errorf("remote(%s) not connected", addr)
	} else {
		return err
	}
}

func (s *TCPServer) onError(m mrun.IModule, err error) {
	if conn, ok := m.UserData().(IConn); ok && conn != nil {
		log.Printf("[D]conn(%s) accur error\n", conn.RemoteAddr())
	}
}
