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
