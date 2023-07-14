package mcommu

import (
	"log"
	"strings"
)

type ICommunicator interface {
	This() ICommunicator
	Protocol() string
	Init(addr string, processor IProcessor, args ...interface{}) error
	// ProtocolInit(addr string, args ...interface{}) error
	Close()
	SendToRemote(addr string, msg interface{}) error
	// RegisterHandler(headerid interface{}, msg interface{}, handler func(conn IConn, req interface{})) error
}

func NewCommunicator(protocol, addr string, maxConnNum, pendingWriteNum, threadnum int, processor IProcessor) ICommunicator {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	if protocol == "" {
		log.Printf("[E]invalid arg\n")
		return nil
	}
	protocol = strings.ToLower(protocol)
	var communicator ICommunicator
	if protocol == "tcpclient" {
		communicator = &TCPClient{}
	} else if protocol == "tcpserver" {
		communicator = &TCPServer{}
	} else if protocol == "udp" {
		communicator = &UDPCommunicator{}
	} else {
		log.Printf("[E]unsurpported protocol(%s)\n", protocol)
		return nil
	}
	err := communicator.Init(addr, processor)
	if err != nil {
		log.Printf("[E]tcpserver init failed:%v\n", err)
		return nil
	}
	return communicator
}
