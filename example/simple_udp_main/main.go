package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

func PbHelloRspHandle(conn mcommu.IConn, req interface{}) {
	if helloRsp, ok := req.(*hellopb.PK_HELLO_RSP); !ok || helloRsp == nil {
		log.Printf("invalid req=%#v\n", req)
	} else {
		log.Printf("rsp=%#v\n", helloRsp)
	}
}

func TestUDP(idx int, addr string, msgprocessor mcommu.IProcessor) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("net dial failed:%v\n", err)
		return
	} else {
		defer conn.Close()
		req := &hellopb.PK_HELLO_REQ{Name: "ranwei" + strconv.Itoa(idx)}
		data, err := msgprocessor.Marshal(req)
		if err != nil {
			log.Printf("processor.Marshal(%#v) failed:%v\n", req, err)
			return
		}
		cnt := 0
		buf := make([]byte, 100)
		for {
			conn.Write(data)
			buf = buf[0:]
			nn, err := conn.Read(buf)
			if err != nil {
				log.Printf("conn.Read failed:%v\n", err)
				return
			}
			headid, msg, leftlen, err := msgprocessor.Unmarshal(buf[:nn])
			if err != nil {
				log.Printf("processor.ParsePkg failed:%v\n", err)
				return
			}
			log.Printf("--leftlen=%d, headid=%v, payload=%#v\n", leftlen, headid, msg)
			time.Sleep(1 * time.Second)
			cnt++
			if cnt > 10 {
				return
			}
		}
	}
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	msgprocessor := &processor.ProtobufProcessor{}
	msgprocessor.RegisterHandler(uint32(hellopb.PK_HELLO_REQ_CMD), &hellopb.PK_HELLO_REQ{}, PbHelloReqHandle)
	msgprocessor.RegisterHandler(uint32(hellopb.PK_HELLO_RSP_CMD), &hellopb.PK_HELLO_RSP{}, PbHelloRspHandle)
	mainudp := mcommu.NewCommunicator("udp", "127.0.0.1:19999", 100, 100, 50, msgprocessor)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit
	mainudp.Close()
	log.Printf("%s Server End!(closing by signal %v)\n", os.Args[0], sig)
}
