package mcommu

import (
	"fmt"
	"log"
	"net"
)

type udpConn struct {
	remoteAddr *net.UDPAddr
	processor  IProcessor
	conn       *net.UDPConn

	communicator *UDPCommunicator
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
