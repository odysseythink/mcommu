package main

import (
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"mlib.com/mcommu"
	"mlib.com/mcommu/example/hellopb"
	"mlib.com/mcommu/processor"
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

func main() {
	userprocessor := &processor.ProtobufProcessor{}
	userprocessor.RegisterHandler(uint32(hellopb.PK_HELLO_REQ_CMD), &hellopb.PK_HELLO_REQ{}, PbHelloReqHandle)
	userprocessor.RegisterHandler(uint32(hellopb.PK_HELLO_RSP_CMD), &hellopb.PK_HELLO_RSP{}, nil)
	s := mcommu.NewCommunicator("tcpserver", "127.0.0.1:19999", 100, 100, 50, userprocessor)
	if s == nil {
		log.Printf("mcommu.NewCommunicator(\"tcpserver\", \":19999\", 100, 100, 50, userprocessor) failed\n")
		return
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit
	s.Close()
	log.Printf("%s Server End!(closing by signal %v)\n", os.Args[0], sig)
}
