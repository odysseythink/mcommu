package hello

import (
	"log"

	"mlib.com/mcommu"
)

type HelloReq struct {
	Name string `json:"name"`
}

type HelloRsp struct {
	Err string `json:"err"`
}

func Hello(conn mcommu.IConn, req interface{}) {
	helloRsp := &HelloRsp{}
	if helloReq, ok := req.(*HelloReq); !ok || helloReq == nil {
		helloRsp.Err = "invalid type"
	} else {
		helloRsp.Err = "hello " + helloReq.Name
	}
	conn.Write(helloRsp)
}

func HelloRspHandler(conn mcommu.IConn, req interface{}) {
	if helloRsp, ok := req.(*HelloRsp); !ok || helloRsp == nil {
		log.Printf("invalid type")
	} else {
		log.Printf("---response %s\n", helloRsp.Err)
	}
}
