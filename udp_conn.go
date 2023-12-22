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

type udpConn struct {
	remoteAddr *net.UDPAddr
	processor  IProcessor
	conn       *net.UDPConn

	communicator *UDPCommunicator
	buf          *bytes.Buffer
	readbuf      *bytes.Buffer
	bufMux       sync.Mutex
	exitWg       sync.WaitGroup
	timeoutTimer *time.Timer
}

func (c *udpConn) Close() {

}
func (c *udpConn) Write(data interface{}) error {
	if c.conn == nil {
		log.Printf("[W]no conn provided\n")
		return fmt.Errorf("[W]no conn provided")
	}
	if c.remoteAddr == nil {
		log.Printf("[W]no remoteAddr provided\n")
		return fmt.Errorf("[W]no remoteAddr provided")
	}
	if c.processor == nil {
		log.Printf("[W]no processor provided\n")
		return fmt.Errorf("[W]no processor provided")
	}
	if c.communicator == nil {
		log.Printf("[W]no communicator provided\n")
		return fmt.Errorf("[W]no communicator provided")
	}

	if data == nil {
		log.Printf("[W]invalid arg\n")
		return fmt.Errorf("invalid arg")
	}

	return c.communicator.SendToAddr(c.remoteAddr, data)
}
func (c *udpConn) RemoteAddr() string {
	if c.remoteAddr == nil {
		log.Printf("[W]no remoteAddr provided\n")
		return ""
	}
	return c.remoteAddr.String()
}
func (c *udpConn) LocalAddr() string {
	if c.conn == nil {
		log.Printf("[W]no conn provided\n")
		return ""
	}
	return c.conn.LocalAddr().String()
}

func (c *udpConn) recieve(data []byte) {
	c.bufMux.Lock()
	c.buf.Write(data)
	c.timeoutTimer.Reset(5000 * time.Millisecond)
	// log.Printf("[D]----recieve=%v\n", data)
	c.bufMux.Unlock()
}

func (c *udpConn) runRecieveLoop(ctx context.Context) {
	periodTimer := time.NewTimer(10 * time.Millisecond)
	defer c.exitWg.Done()
	for {
		select {
		case <-periodTimer.C:
			if c.recieveProcess() != nil {
				return
			}
			periodTimer.Reset(10 * time.Millisecond)
		case <-c.timeoutTimer.C:
			// log.Printf("[D]----timeout")
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *udpConn) recieveProcess() error {
	c.bufMux.Lock()
	defer c.bufMux.Unlock()
	for c.buf.Len() >= c.processor.HeaderLen() {
		c.readbuf.Reset()
		c.buf.Read(c.readbuf.Bytes()[:c.processor.HeaderLen()])
		msgid, msglen, err := c.processor.ParseHeader(c.readbuf.Bytes()[:c.processor.HeaderLen()])
		if err != nil {
			log.Printf("[E]processor.ParseHeader failed: %v\n", err)
			return fmt.Errorf("processor.ParseHeader failed: %v", err)
		}
		if msglen+c.processor.HeaderLen() > c.buf.Cap() {
			log.Printf("[E]package len(%d) is beyond processor cap(%d)\n", msglen+c.processor.HeaderLen(), c.buf.Cap())
			return fmt.Errorf("package len(%d) is beyond processor cap(%d)", msglen+c.processor.HeaderLen(), c.buf.Cap())
		}
		// log.Printf("[D]---msglen=%d \n", msglen)

		if msglen > 0 {
			datalen := c.buf.Len()
			if msglen > c.buf.Len() {
				log.Printf("[W]msglen(%d) not match recieve buffer len(%d)\n", msglen, c.buf.Len())
				nn, err := c.buf.Read(c.readbuf.Bytes()[c.processor.HeaderLen() : c.processor.HeaderLen()+datalen])
				if nn != datalen || err != nil {
					log.Printf("[E]read buffer failed: %v\n", err)
					return fmt.Errorf("read buffer failed: %v", err)
				}
				c.buf.Write(c.readbuf.Bytes()[:c.processor.HeaderLen()+datalen])
				return nil
			}
			nn, err := c.buf.Read(c.readbuf.Bytes()[c.processor.HeaderLen() : c.processor.HeaderLen()+msglen])
			if err != nil || nn != msglen {
				log.Printf("[E]read buf failed: %v\n", err)
				return fmt.Errorf("read buf failed: %v", err)
			}
			// log.Printf("[D]---c.readbuf=%v \n", c.readbuf[:msglen])

			msg, err := c.processor.Unmarshal(msgid, c.readbuf.Bytes()[c.processor.HeaderLen():c.processor.HeaderLen()+msglen])
			if err != nil {
				log.Printf("[E]processor.Unmarshal(%#v) failed: %v\n", msgid, err)
				return fmt.Errorf("processor.Unmarshal(%#v) failed: %v", msgid, err)
			}

			msgfunc, err := c.processor.Route(msgid, msg)
			if err != nil {
				log.Printf("[W]processor.Route failed: %v\n", err)
			} else {
				mrun.WorkerSubmit(func() {
					msgfunc(c, msg)
				})
			}
		} else {
			msg, err := c.processor.Unmarshal(msgid, nil)
			if err != nil {
				log.Printf("[W]processor.Unmarshal(%#v) failed: %v\n", msgid, err)
				return fmt.Errorf("processor.Unmarshal(%#v) failed: %v", msgid, err)
			}

			msgfunc, err := c.processor.Route(msgid, msg)
			if err != nil {
				log.Printf("[W]processor.Route failed: %v\n", err)
			} else {
				mrun.WorkerSubmit(func() {
					msgfunc(c, msg)
				})
			}
		}
	}
	return nil
}
