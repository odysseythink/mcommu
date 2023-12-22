package main

import (
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"mlib.com/mcommu"
	"mlib.com/mcommu/example/hellopb"
	"mlib.com/mcommu/processor"
)

func TestClient(idx int, addr string, msgprocessor mcommu.IProcessor) {
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
		log.Printf("[D]write data:%#v\n", data)

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
	msgprocessor.RegisterHandler(uint32(hellopb.PK_HELLO_REQ_CMD), &hellopb.PK_HELLO_REQ{}, nil)
	msgprocessor.RegisterHandler(uint32(hellopb.PK_HELLO_RSP_CMD), &hellopb.PK_HELLO_RSP{}, nil)
	var wg sync.WaitGroup
	for iLoop := 0; iLoop < 1; iLoop++ {
		// wg.Add(1)
		// go func(idx int) {
		TestClient(iLoop, "127.0.0.1:19999", msgprocessor)
		// wg.Done()
		// }(iLoop)
	}
	wg.Wait()

}
