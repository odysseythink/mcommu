package mcommu

import (
	"log"
	"net"
	"runtime"
	"strconv"
)

func IsNetAddrValid(addr string) bool {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[E]addr(%s)have the invalid format:%v\n", addr, err)
		return false
	}
	if ip := net.ParseIP(host); ip == nil {
		log.Printf("[E]net.ParseIP(%s) failed:%v\n", host, err)
		return false
	}

	iport, err := strconv.Atoi(port)
	if err != nil {
		log.Printf("[E]port(%s) convert to int failed:%v\n", port, err)
		return false
	}

	if iport < 0 || iport > 65535 {
		log.Printf("[E]port(%d) must in range:0~65535\n", iport)
		return false
	}
	return true
}

var ms runtime.MemStats

func DebugMem() {
	runtime.GC()
	runtime.ReadMemStats(&ms)
	// fmt.Printf("-----%+v\n", m)
	// fmt.Printf("-----os %d\n", m.Sys)
	log.Printf("------sys:%d Alloc:%d(bytes) HeapAlloc:%d(bytes) HeapIdle:%d(bytes) HeapReleased:%d(bytes)\n", ms.Sys, ms.Alloc, ms.HeapAlloc, ms.HeapIdle, ms.HeapReleased)
}
