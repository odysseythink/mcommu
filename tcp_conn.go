package mcommu

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"mlib.com/mrun"
)

type tcpConnIOBase struct {
	conn      net.Conn
	processor IProcessor
	parent    IConn
}

func (c *tcpConnIOBase) UserData() interface{} {
	return c.parent
}

func (c *tcpConnIOBase) Init(args ...interface{}) error {
	// conn net.Conn, processor IProcessor, parent IConn) error {
	if len(args) != 3 {
		log.Printf("[E]args(conn *net.UDPConn, processor IProcessor, parent *udpEndpoint) is needed\n")
		return fmt.Errorf("args(conn *net.UDPConn, processor IProcessor, parent *udpEndpoint) is needed")
	}
	log.Printf("......args=%#v\n", args)
	if conn, ok := args[0].(net.Conn); !ok || conn == nil {
		log.Printf("[E]args[0](%#v) must be a valid net.Conn\n", args[0])
		return fmt.Errorf("args[0](%#v) must be a valid net.Conn", args[0])
	} else {
		if processor, ok := args[1].(IProcessor); !ok || processor == nil {
			log.Printf("[E]args[1](%#v) must be a valid IProcessor\n", args[1])
			return fmt.Errorf("args[1](%#v) must be a valid IProcessor", args[1])
		} else {
			if parent, ok := args[2].(IConn); !ok || parent == nil {
				log.Printf("[E]args[2](%#v) must be a valid IConn\n", args[2])
				return fmt.Errorf("args[2](%#v) must be a valid IConn", args[2])
			} else {
				c.conn = conn
				c.processor = processor
				c.parent = parent
				return nil
			}
		}
	}
}

func (c *tcpConnIOBase) Destroy() {
}

type tcpConnWriter struct {
	tcpConnIOBase
	writeCh        chan []byte
	writeChCondMux sync.Mutex
	writeChCond    *sync.Cond
}

// b must not be modified by the others goroutines
func (w *tcpConnWriter) write(data []byte) error {
	if w.conn == nil {
		log.Printf("[W]no conn provided")
		return fmt.Errorf("[W]no conn provided")
	}
	if w.writeCh == nil {
		w.writeCh = make(chan []byte, 1024)
	}
	if w.writeChCond == nil {
		w.writeChCond = sync.NewCond(&w.writeChCondMux)
	}
	if data == nil {
		log.Printf("[W]invalid arg")
		return fmt.Errorf("invalid arg")
	}
	for {
		select {
		// 写数据b
		case w.writeCh <- data:
			return nil
		default:
			log.Printf("[W]channel full, retry")
			w.writeChCond.Wait()
		}
	}
}

func (w *tcpConnWriter) Write(data interface{}) error {
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
	pkg, err := w.processor.Marshal(data)
	if err != nil {
		log.Printf("[W]processor.Marshal(%#v) failed:%v\n", data, err)
		return fmt.Errorf("[W]processor.Marshal(%#v) failed:%v", data, err)
	}
	if pkg == nil {
		log.Printf("[W]processor.Marshal(%#v) return nil package\n", data)
		return fmt.Errorf("[W]processor.Marshal(%#v) return nil package", data)
	}
	w.write(pkg)
	return nil
}

func (w *tcpConnWriter) RunOnce(context.Context) error {
	if w.parent == nil {
		log.Printf("[W]no IConn provided\n")
		return fmt.Errorf("no IConn provided")
	}

	if w.conn == nil {
		log.Printf("[W]no conn provided\n")
		return fmt.Errorf("no conn provided")
	}
	if w.writeCh == nil {
		w.writeCh = make(chan []byte, 1024)
	}
	if w.writeChCond == nil {
		w.writeChCond = sync.NewCond(&w.writeChCondMux)
	}
	timeout := time.NewTimer(time.Microsecond * 10)

	select {
	case data := <-w.writeCh:
		w.writeChCond.Signal()
		if data != nil {
			_, err := w.conn.Write(data)
			if err != nil {
				log.Printf("[E]conn write failed:%v\n", err)
				// w.userProcessor.OnClose(w.parent)
				return fmt.Errorf("conn write failed:%v", err)
			}
		}
	case <-timeout.C:
		return nil
	}
	return nil
}

type tcpConnReader struct {
	tcpConnIOBase

	leftData    []byte
	leftDataBuf *bytes.Buffer
	buf         []byte
	readbuf     []byte
	startIdx    int
	copyLen     int
}

func (r *tcpConnReader) RunOnce(context.Context) error {
	if r.parent == nil {
		log.Printf("[W]no tcpConn provided\n")
		return fmt.Errorf("no tcpConn provided")
	}
	if r.conn == nil {
		log.Printf("[W]no conn provided\n")
		return fmt.Errorf("no conn provided")
	}
	if r.processor == nil {
		log.Printf("[W]no processor provided\n")
		return fmt.Errorf("no processor provided")
	}
	if r.leftData == nil {
		r.leftData = make([]byte, 0, 2048)
		r.leftDataBuf = bytes.NewBuffer(r.leftData)
		r.leftDataBuf.Reset()
		r.startIdx = 0
		r.copyLen = 0
	}
	if r.buf == nil {
		r.buf = make([]byte, 1024)
	}
	if r.readbuf == nil {
		r.readbuf = make([]byte, 1024)
	}

	// DebugMem()
	r.buf = r.buf[0:]
	nn, err := r.conn.Read(r.buf)
	if err != nil {
		log.Printf("[E]read message failed: %v\n", err)
		return fmt.Errorf("read message failed: %v", err)
	}
	log.Printf("[D]%s from %s read %d bytes \n", r.conn.LocalAddr().String(), r.conn.RemoteAddr().String(), nn)
	r.startIdx = 0
	for nn > 0 {
		if r.leftDataBuf.Cap()-r.leftDataBuf.Len() >= nn {
			r.copyLen = nn
		} else {
			r.copyLen = r.leftDataBuf.Cap() - r.leftDataBuf.Len()
		}
		r.leftDataBuf.Write(r.buf[r.startIdx : r.startIdx+r.copyLen])
		r.startIdx += r.copyLen
		nn -= r.copyLen
		headid, msg, leftlen, err := r.processor.Unmarshal(r.leftDataBuf.Bytes())
		if err != nil {
			log.Printf("[E]processor.UnMarshal failed: %v\n", err)
			return fmt.Errorf("processor.UnMarshal failed: %v", err)
		}
		msgfunc, err := r.processor.Route(headid, msg)
		if err != nil {
			log.Printf("[W]processor.Route failed: %v\n", err)
		} else {
			mrun.WorkerSubmit(func() {
				msgfunc(r.parent, msg)
			})
		}

		r.readbuf = r.readbuf[0 : r.leftDataBuf.Len()-leftlen]
		r.leftDataBuf.Read(r.readbuf)
	}
	return nil
}

type tcpConn struct {
	tcpConnReader
	tcpConnWriter
	ioMgr     mrun.ModuleMgr
	conn      net.Conn
	processor IProcessor
	closeOnce sync.Once
}

// func newTCPConn(conn net.Conn, processor IProcessor) *tcpConn {
// 	if processor == nil || conn == nil {
// 		log.Printf("[E]both processor and net.Conn are needed")
// 		return nil
// 	}

// 	tcpconn := &tcpConn{}
// 	tcpconn.conn = conn
// 	tcpconn.processor = processor
// 	tcpconn.ioMgr.Register(&tcpconn.tcpConnReader, nil, conn, processor, tcpconn)
// 	tcpconn.ioMgr.Register(&tcpconn.tcpConnWriter, nil, conn, processor, tcpconn)
// 	err := tcpconn.Init()
// 	if err != nil {
// 		log.Printf("[E]tcpconn init failed:%v\n", err)
// 		return nil
// 	}

// 	return tcpconn
// }

func (c *tcpConn) Init(args ...interface{}) error {
	// log.Printf("......args=%#v\n", args)
	// conn net.Conn, processor IProcessor
	if len(args) != 2 {
		log.Printf("[E]args(conn net.Conn, processor IProcessor) is needed\n")
		return fmt.Errorf("args(conn net.Conn, processor IProcessor) is needed")
	}
	if conn, ok := args[0].(net.Conn); !ok || conn == nil {
		log.Printf("[E]args[0](%#v) must be a valid net.Conn\n", args[0])
		return fmt.Errorf("args[0](%#v) must be a valid net.Conn", args[0])
	} else {
		if processor, ok := args[1].(IProcessor); !ok || processor == nil {
			log.Printf("[E]args[1](%#v) must be a valid IProcessor\n", args[1])
			return fmt.Errorf("args[1](%#v) must be a valid IProcessor", args[1])
		} else {
			c.conn = conn
			c.processor = processor
			c.ioMgr.Register(&c.tcpConnReader, []mrun.ModuleMgrOption{mrun.NewModuleErrorOption(c.onError)}, conn, processor, c)
			c.ioMgr.Register(&c.tcpConnWriter, []mrun.ModuleMgrOption{mrun.NewModuleErrorOption(c.onError)}, conn, processor, c)
			err := c.ioMgr.Init()
			if err != nil {
				log.Printf("[E]tcpconn init failed:%v\n", err)
				return nil
			}
		}
	}
	return nil
}

func (c *tcpConn) RemoteAddr() string {
	if c.conn == nil {
		return ""
	}
	return c.conn.RemoteAddr().String()
}

func (c *tcpConn) LocalAddr() string {
	if c.conn == nil {
		return ""
	}
	return c.conn.LocalAddr().String()
}

func (c *tcpConn) Destroy() {
	c.closeOnce.Do(func() {
		if c.conn != nil {
			// log.Printf("[D]remote(%s) closing\n", c.RemoteAddr())
			c.conn.(*net.TCPConn).SetLinger(0)
			c.conn.Close()
		}
		c.ioMgr.Destroy()
		c.conn = nil
		log.Printf("[D]close done")
	})
}

func (c *tcpConn) Close() {
	c.Destroy()
}

func (c *tcpConn) RunOnce(context.Context) error {
	if c.conn == nil {
		return fmt.Errorf("conn already closed")
	}
	return nil
}

func (c *tcpConn) onError(m mrun.IModule, err error) {
	c.Destroy()
}

func (c *tcpConn) UserData() interface{} {
	return c
}
