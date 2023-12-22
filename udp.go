package mcommu

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"mlib.com/mrun"
)

type UDPCommunicator struct {
	addr      string
	conn      *net.UDPConn
	processor IProcessor
	udpCommunicatorReader
	ioMgr     mrun.ModuleMgr
	closeOnce sync.Once
}

func (c *UDPCommunicator) This() ICommunicator {
	return c
}
func (c *UDPCommunicator) Protocol() string {
	return "udp"
}
func (c *UDPCommunicator) Init(addr string, processor IProcessor, args ...interface{}) error {
	if addr == "" || processor == nil {
		log.Printf("[E]inavlid arg\n")
		return fmt.Errorf("[E]inavlid arg")
	}
	if !IsNetAddrValid(addr) {
		log.Printf("[E]inavlid addr(%s)\n", addr)
		return fmt.Errorf("[E]inavlid addr(%s)", addr)
	}
	c.processor = processor
	c.addr = addr
	var err error
	udpaddr, err := net.ResolveUDPAddr("udp", c.addr)
	if err != nil {
		log.Printf("[E]net.ResolveUDPAddr(\"udp\", %s) failed:%v\n", c.addr, err)
		return fmt.Errorf("net.ResolveUDPAddr(\"udp\", %s) failed:%v", c.addr, err)
	}

	c.conn, err = net.ListenUDP("udp", udpaddr)
	if err != nil {
		log.Printf("[E]net.ListenUDP(\"udp\", %#v) failed:%v\n", udpaddr, err)
		return fmt.Errorf("net.ListenUDP(\"udp\", %#v) failed:%v", udpaddr, err)
	}
	c.ioMgr.Register(&c.udpCommunicatorReader, []mrun.ModuleMgrOption{mrun.NewModuleErrorOption(c.onError)}, c.conn, processor, c)
	// c.ioMgr.Register(&c.udpCommunicatorWriter, []mrun.ModuleMgrOption{mrun.NewModuleErrorOption(c.onError)}, c.conn, processor, c)
	err = c.ioMgr.Init()
	if err != nil {
		log.Printf("[E]tcpconn init failed:%v\n", err)
		c.conn.Close()
		return nil
	}

	return nil
}

// ProtocolInit(addr string, args ...interface{}) error
func (c *UDPCommunicator) Close() {
	c.closeOnce.Do(func() {
		if c.conn != nil {
			c.conn.Close()
		}
		c.ioMgr.Destroy()
	})
}

func (c *UDPCommunicator) onError(m mrun.IModule, err error) {
}

func (w *UDPCommunicator) SendToRemote(addr string, data interface{}) error {
	if w.conn == nil {
		log.Printf("[W]no conn provided\n")
		return fmt.Errorf("[W]no conn provided")
	}
	if w.processor == nil {
		log.Printf("[W]no processor provided\n")
		return fmt.Errorf("[W]no processor provided")
	}

	if data == nil {
		log.Printf("[W]invalid arg\n")
		return fmt.Errorf("invalid arg")
	}
	if !IsNetAddrValid(addr) {
		log.Printf("[W]invalid addr(%s)\n", addr)
		return fmt.Errorf("invalid addr(%s)", addr)
	}

	pkg, err := w.processor.Marshal(data)
	if err != nil {
		log.Printf("[W]userProcessor.Marshal(%#v) failed:%v\n", data, err)
		return fmt.Errorf("[W]userProcessor.Marshal(%#v) failed:%v", data, err)
	}
	if pkg == nil {
		log.Printf("[W]userProcessor.Marshal(%#v) return nil package\n", data)
		return fmt.Errorf("[W]userProcessor.Marshal(%#v) return nil package", data)
	}
	tmps := strings.Split(addr, ":")
	port, _ := strconv.Atoi(tmps[1])
	w.sendTo(&net.UDPAddr{IP: net.ParseIP(tmps[0]), Port: port}, pkg)
	return nil
}

func (w *UDPCommunicator) SendToAddr(addr *net.UDPAddr, data interface{}) error {
	if w.conn == nil {
		log.Printf("[W]no conn provided\n")
		return fmt.Errorf("[W]no conn provided")
	}
	if w.processor == nil {
		log.Printf("[W]no processor provided\n")
		return fmt.Errorf("[W]no processor provided")
	}

	if data == nil || addr == nil {
		log.Printf("[W]invalid arg\n")
		return fmt.Errorf("invalid arg")
	}

	pkg, err := w.processor.Marshal(data)
	if err != nil {
		log.Printf("[W]userProcessor.Marshal(%#v) failed:%v\n", data, err)
		return fmt.Errorf("[W]userProcessor.Marshal(%#v) failed:%v", data, err)
	}
	if pkg == nil {
		log.Printf("[W]userProcessor.Marshal(%#v) return nil package\n", data)
		return fmt.Errorf("[W]userProcessor.Marshal(%#v) return nil package", data)
	}
	w.sendTo(addr, pkg)
	return nil
}

// b must not be modified by the others goroutines
func (w *UDPCommunicator) sendTo(addr *net.UDPAddr, data []byte) error {
	if w.conn == nil {
		log.Printf("[W]no conn provided")
		return fmt.Errorf("[W]no conn provided")
	}

	if data == nil || addr == nil {
		log.Printf("[W]invalid arg")
		return fmt.Errorf("invalid arg")
	}

	_, err := w.conn.WriteToUDP(data, addr)
	if err != nil {
		log.Printf("[E]conn write failed:%v\n", err)
		return fmt.Errorf("conn write failed:%v", err)
	}
	return nil
}

type udpCommunicatorBaseIO struct {
	conn      *net.UDPConn
	processor IProcessor
	parent    *UDPCommunicator
}

func (c *udpCommunicatorBaseIO) Init(args ...interface{}) error {
	if len(args) != 3 {
		log.Printf("[E]args(conn *net.UDPConn, processor IProcessor, parent *udpEndpoint) is needed\n")
		return fmt.Errorf("args(conn *net.UDPConn, processor IProcessor, parent *udpEndpoint) is needed")
	}
	if conn, ok := args[0].(*net.UDPConn); !ok || conn == nil {
		log.Printf("[E]args[0](%#v) must be a valid net.UDPConn pointer\n", args[0])
		return fmt.Errorf("args[0](%#v) must be a valid net.UDPConn pointer", args[0])
	} else {
		if processor, ok := args[1].(IProcessor); !ok || processor == nil {
			log.Printf("[E]args[1](%#v) must be a valid IProcessor\n", args[1])
			return fmt.Errorf("args[1](%#v) must be a valid IProcessor", args[1])
		} else {
			if parent, ok := args[2].(*UDPCommunicator); !ok || parent == nil {
				log.Printf("[E]args[2](%#v) must be a valid udpEndpoint pointer\n", args[2])
				return fmt.Errorf("args[2](%#v) must be a valid udpEndpoint pointer", args[2])
			} else {
				c.conn = conn
				c.processor = processor
				c.parent = parent
				return nil
			}
		}
	}
}

func (c *udpCommunicatorBaseIO) Destroy() {
}

func (c *udpCommunicatorBaseIO) UserData() interface{} {
	return c.parent
}

type udpCommunicatorReader struct {
	udpCommunicatorBaseIO

	bufferPool sync.Pool
	readBuf    []byte
	clients    sync.Map
}

func (r *udpCommunicatorReader) RunOnce(ctx context.Context) error {
	if r.parent == nil {
		log.Printf("[W]no UDPCommunicator provided")
		return fmt.Errorf("no UDPCommunicator provided")
	}
	if r.conn == nil {
		log.Printf("[W]no conn provided\n")
		return fmt.Errorf("no conn provided")
	}
	if r.processor == nil {
		log.Printf("[W]no processor provided\n")
		return fmt.Errorf("no processor provided")
	}
	if r.bufferPool.New == nil {
		r.bufferPool.New = func() interface{} {
			return bytes.NewBuffer(make([]byte, 512))
		}
	}

	if r.readBuf == nil {
		r.readBuf = make([]byte, 512)
	}

	// DebugMem()
	nn, rAddr, err := r.conn.ReadFromUDP(r.readBuf)
	if err != nil {
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return nil
		}
		log.Printf("[E]read message failed: %v\n", err)
		return fmt.Errorf("read message failed: %v", err)
	}
	if nn > 0 {
		// log.Printf("[D]%s from %s read %d bytes \n", r.conn.LocalAddr().String(), rAddr.String(), nn)
		var cli *udpConn
		if val, ok := r.clients.Load(rAddr.String()); !ok {
			cli = &udpConn{}
			cli.remoteAddr = rAddr
			cli.processor = r.processor
			cli.conn = r.conn
			cli.communicator = r.parent
			cli.buf = r.bufferPool.Get().(*bytes.Buffer)
			cli.buf.Reset()
			cli.timeoutTimer = time.NewTimer(5000 * time.Millisecond)
			cli.readbuf = r.bufferPool.Get().(*bytes.Buffer)
			r.clients.Store(rAddr.String(), cli)
			cli.exitWg.Add(1)
			go cli.runRecieveLoop(ctx)
			go func(c *udpConn) {
				c.exitWg.Wait()
				r.bufferPool.Put(c.buf)
				r.bufferPool.Put(c.readbuf)
				r.clients.Delete(cli.remoteAddr.String())
			}(cli)
		} else {
			cli = val.(*udpConn)
		}
		cli.recieve(r.readBuf[:nn])
	}

	return nil
}
