package mcommu

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"mlib.com/mrun"
)

type TCPClient struct {
	ctx             context.Context
	wg              sync.WaitGroup
	ctxCancelFunc   context.CancelFunc
	addr            string
	tcpConnMgr      mrun.ModuleMgr
	processor       IProcessor
	connnum         int
	lastSendChannel int
	idleList        *list.List
	threadnum       int
}

func (c *TCPClient) Init(addr string, processor IProcessor, args ...interface{}) error {
	if addr == "" || processor == nil {
		log.Printf("[E]inavlid arg\n")
		return fmt.Errorf("[E]inavlid arg")
	}
	if !IsNetAddrValid(addr) {
		log.Printf("[E]inavlid addr(%s)\n", addr)
		return fmt.Errorf("[E]inavlid addr(%s)", addr)
	}
	c.addr = addr
	if len(args) > 0 {
		if threadnum, ok := args[0].(int); !ok || threadnum <= 0 {
			log.Printf("[E]args[0](%#v) must be a valid integer to define threadnum\n", args[0])
			return fmt.Errorf("[E]args[0](%#v) must be a valid integer to define threadnum", args[0])
		} else {
			c.threadnum = threadnum
		}
		if len(args) > 1 {
			if connnum, ok := args[1].(int); !ok || connnum <= 0 {
				log.Printf("[E]args[1](%#v) must be a valid integer to define connnum\n", args[1])
				return fmt.Errorf("[E]args[1](%#v) must be a valid integer to define connnum", args[1])
			} else {
				c.connnum = connnum
			}
		} else {
			c.connnum = 10
		}
	} else {
		c.threadnum = 50
		c.connnum = 10
	}
	c.processor = processor
	var err error
	err = c.tcpConnMgr.Init()
	if err != nil {
		log.Printf("[E]tcpConnMgr init failed:%v\n", err)
		return fmt.Errorf("tcpConnMgr init failed:%v", err)
	}
	c.ctx, c.ctxCancelFunc = context.WithCancel(context.Background())
	c.idleList = list.New()
	c.run()
	return nil
}

func (c *TCPClient) This() ICommunicator {
	return c
}

func (c *TCPClient) Protocol() string {
	return "tcpclient"
}

func (c *TCPClient) Close() {
	if c.ctxCancelFunc != nil {
		c.ctxCancelFunc()
	}
	c.tcpConnMgr.Destroy()
	c.wg.Wait()
}

func (c *TCPClient) SendToRemote(addr string, msg interface{}) error {
	if c.idleList.Len() == 0 {
		for iLoop := 0; iLoop < c.connnum; iLoop++ {
			c.idleList.PushBack(strconv.Itoa(iLoop))
		}
	}
begin:
	if c.idleList.Len() == 0 {
		log.Printf("[E]no valid conn\n")
		return fmt.Errorf("no valid conn")
	}
	idle := c.idleList.Front().Value.(string)
	c.idleList.Remove(c.idleList.Front())
	log.Printf("......SendToRemote %s\n", idle)
	mods := c.tcpConnMgr.GetModulesByAlias(idle)
	if len(mods) > 0 {
		if conn, ok := mods[0].UserData().(IConn); !ok && conn == nil {
			log.Printf("[W]current(alias=%s) is  invalid, retry\n", idle)
			goto begin
		} else {
			return conn.Write(msg)
		}
	} else {
		log.Printf("[W]can't get module by(alias=%s), retry\n", idle)
		goto begin
	}
}

func (c *TCPClient) run() {
	if c.ctx == nil {
		c.ctx, c.ctxCancelFunc = context.WithCancel(context.Background())
	}
	for iLoop := 0; iLoop < c.connnum; iLoop++ {
		if c.tcpConnMgr.ModuleNum() < c.connnum {
			log.Printf("[D]add %d\n", iLoop)
			mods := c.tcpConnMgr.GetModulesByAlias(strconv.Itoa(iLoop))
			if mods == nil {
				conn, err := net.Dial("tcp", c.addr)
				if err != nil {
					log.Printf("[E]net dial failed:%v\n", err)
				} else {
					tcpconn := &tcpConn{}
					err = c.tcpConnMgr.Register(tcpconn, []mrun.ModuleMgrOption{mrun.NewModuleAliasOption(strconv.Itoa(iLoop))}, conn, c.processor)
					if err != nil {
						log.Printf("[W]conn retister failed:%v\n", err)
						conn.Close()
						continue
					}
					c.idleList.PushBack(strconv.Itoa(iLoop))
				}
			}
		}

		c.wg.Add(1)
		go func(idx int) {
			c.runEvery(idx)
			c.wg.Done()
		}(iLoop)
	}
}

func (c *TCPClient) runEvery(idx int) {
	runTimer := time.NewTimer(1 * time.Nanosecond)
	defer runTimer.Stop()
LOOP:
	for {
		select {
		case <-c.ctx.Done():
			log.Printf("[D]context done\n")
			break LOOP
		case <-runTimer.C:
			mods := c.tcpConnMgr.GetModulesByAlias(strconv.Itoa(idx))
			if mods == nil {
				conn, err := net.Dial("tcp", c.addr)
				if err != nil {
					log.Printf("[E]net dial failed:%v\n", err)
					// time.Sleep(1 * time.Second)
					runTimer.Reset(1 * time.Second)
				} else {
					tcpconn := &tcpConn{}
					err = c.tcpConnMgr.Register(tcpconn, []mrun.ModuleMgrOption{mrun.NewModuleAliasOption(strconv.Itoa(idx))}, conn, c)
					if err != nil {
						log.Printf("[W]conn retister failed:%v\n", err)
						conn.Close()
						continue
					}
					c.idleList.PushBack(strconv.Itoa(idx))
					runTimer.Reset(100 * time.Millisecond)
				}
			}
		}
	}
}
