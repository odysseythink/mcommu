package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"mlib.com/mcommu"
	"mlib.com/mcommu/example/hellopb"
	"mlib.com/mcommu/processor"
	"mlib.com/mrun"
)

var (
	sucTimes    int32 = 0
	concurrency       = 5000
)

func PbHelloReqHandle(conn mcommu.IConn, req interface{}) {
	rsp := &hellopb.PK_HELLO_RSP{}
	if helloReq, ok := req.(*hellopb.PK_HELLO_REQ); !ok || helloReq == nil {
		rsp.Errmsg = "invalid type"
	} else {
		rsp.Errmsg = "hello " + helloReq.Name
	}
	conn.Write(rsp)
}

func PbHelloRspHandle(conn mcommu.IConn, req interface{}) {
	if helloRsp, ok := req.(*hellopb.PK_HELLO_RSP); !ok || helloRsp == nil {
		log.Printf("invalid req=%#v\n", req)
	}
}

type subudp struct {
	processor    mcommu.IProcessor
	sendTime     time.Time
	communicator mcommu.ICommunicator
	cnt          int
}

func (u *subudp) Init(args ...interface{}) error {
	if len(args) != 2 {
		log.Printf("[E]args(conn *net.UDPConn, processor IProcessor, parent *udpEndpoint) is needed\n")
		return fmt.Errorf("args(conn *net.UDPConn, processor IProcessor, parent *udpEndpoint) is needed")
	}
	if port, ok := args[0].(int); !ok || port <= 0 || port > 65535 {
		log.Printf("[E]args[0](%#v) must be a valid IProcessor\n", args[0])
		return fmt.Errorf("args[0](%#v) must be a valid IProcessor", args[0])
	} else {
		if processor, ok := args[1].(mcommu.IProcessor); !ok || processor == nil {
			log.Printf("[E]args[1](%#v) must be a valid IProcessor\n", args[1])
			return fmt.Errorf("args[1](%#v) must be a valid IProcessor", args[1])
		} else {
			u.processor = processor
			u.communicator = mcommu.NewCommunicator("udp", "127.0.0.1:"+strconv.Itoa(port), 100, 100, 50, u.processor)
			if u.communicator == nil {
				log.Printf("[E]can't new Communicator\n")
				return fmt.Errorf("can't new Communicator")
			}
		}
	}

	u.sendTime = time.Now().Add(1 * time.Second)

	return nil
}

func (u *subudp) RunOnce(context.Context) error {
	now := time.Now()
	if now.After(u.sendTime) {
		u.communicator.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(u.cnt)})
		u.cnt++
		u.sendTime = now.Add(1 * time.Second)
	}
	if u.cnt >= 10 {
		atomic.AddInt32(&sucTimes, int32(1))
		return fmt.Errorf("beyond max times, exit")
	}
	return nil
}

func (u *subudp) Destroy() {
	u.communicator.Close()
}
func (u *subudp) UserData() interface{} {
	return nil
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	msgprocessor := &processor.ProtobufProcessor{}
	msgprocessor.RegisterHandler(uint32(hellopb.PK_HELLO_REQ_CMD), &hellopb.PK_HELLO_REQ{}, PbHelloReqHandle)
	msgprocessor.RegisterHandler(uint32(hellopb.PK_HELLO_RSP_CMD), &hellopb.PK_HELLO_RSP{}, PbHelloRspHandle)

	subs := mrun.ModuleMgr{}
	for iLoop := 29999; iLoop < 29999+concurrency; iLoop++ {
		subs.Register(&subudp{}, nil, iLoop, msgprocessor)
	}
	err := subs.Init()
	if err != nil {
		log.Printf("subs.Init() failed:%v\n", err)
		return
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	timer := time.NewTimer(100 * time.Millisecond)
	for {
		select {
		case sig := <-quit:
			subs.Destroy()
			log.Printf("%s Server End!(closing by signal %v)\n", os.Args[0], sig)
			return
		case <-timer.C:
			if subs.ModuleNum() == 0 {
				subs.Destroy()
				log.Printf("%s Server End!(all worker exit)\n", os.Args[0])
				if sucTimes == int32(concurrency) {
					log.Printf("all subudp success\n")
				}
				return
			}
			timer.Reset(100 * time.Millisecond)
		}
	}

}
