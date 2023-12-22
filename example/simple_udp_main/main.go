package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
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
			msgid, msglen, err := msgprocessor.ParseHeader(buf[:msgprocessor.HeaderLen()])
			if err != nil {
				log.Printf("processor.ParseHeader failed:%v\n", err)
				return
			}
			log.Printf("--msglen=%d, headid=%v\n", msglen, msgid)
			if nn != (msgprocessor.HeaderLen() + msglen) {
				log.Printf("msg len not match, wanted(%d), but real(%d)\n", msgprocessor.HeaderLen()+msglen, nn)
				return
			}
			msg, err := msgprocessor.Unmarshal(msgid, buf[msgprocessor.HeaderLen():nn])
			if err != nil {
				log.Printf("processor.Unmarshal failed:%v\n", err)
				return
			}
			log.Printf("--msg=%#v\n", msg)
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
