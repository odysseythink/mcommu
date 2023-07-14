package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"mlib.com/mcommu"
	"mlib.com/mcommu/example/hellopb"
	"mlib.com/mcommu/processor"
)

func PbHelloRspHandle(conn mcommu.IConn, req interface{}) {
	if helloRsp, ok := req.(*hellopb.PK_HELLO_RSP); !ok || helloRsp == nil {
		log.Printf("invalid type")
	} else {
		log.Printf("---response %s\n", helloRsp.Errmsg)
	}
}

func main() {
	msgprocessor := &processor.ProtobufProcessor{}
	msgprocessor.RegisterHandler(uint32(hellopb.PK_HELLO_REQ_CMD), &hellopb.PK_HELLO_REQ{}, nil)
	msgprocessor.RegisterHandler(uint32(hellopb.PK_HELLO_RSP_CMD), &hellopb.PK_HELLO_RSP{}, PbHelloRspHandle)
	// var wg sync.WaitGroup
	for iLoop := 0; iLoop < 10000; iLoop++ {
		s := mcommu.NewCommunicator("tcpclient", "127.0.0.1:19999", 100, 100, 50, msgprocessor)

		// time.Sleep(1 * time.Second)
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+0)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+1)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+2)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+3)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+4)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+5)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+6)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+7)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+8)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+9)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+10)})
		s.SendToRemote("127.0.0.1:19999", &hellopb.PK_HELLO_REQ{Name: "hello" + strconv.Itoa(iLoop+11)})
		// time.Sleep(3 * time.Second)
		s.Close()
	}
	// wg.Wait()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit

	log.Printf("%s Server End!(closing by signal %v)\n", os.Args[0], sig)
}
