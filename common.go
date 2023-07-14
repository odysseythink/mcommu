package mcommu

import (
	"log"
	"net"
	"runtime"
	"strconv"
	"strings"
)

func IsNetAddrValid(addr string) bool {
	addr = strings.ReplaceAll(addr, " ", "")
	if addr == "" {
		log.Printf("[E]invalid arg\n")
		return false
	}
	tmps := strings.Split(addr, ":")
	if len(tmps) != 2 {
		log.Printf("[E]addr(%s) must have the format(ip:port)\n", addr)
		return false
	}
	if tmps[0] != "" {
		_, err := net.ResolveIPAddr("ip", tmps[0])
		if err != nil {
			log.Printf("[E]net.ResolveIPAddr(\"ip\", %s) failed:%v\n", tmps[0], err)
			return false
		}
	}

	port, err := strconv.Atoi(tmps[1])
	if err != nil {
		log.Printf("[E]port(%s) convert to int failed\n", tmps[1])
		return false
	}

	if port < 0 || port > 65535 {
		log.Printf("[E]port(%s) must in range:0~65535\n", tmps[1])
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
