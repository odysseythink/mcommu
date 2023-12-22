package processor

import (
	"encoding/binary"
	"fmt"
	"log"
	"reflect"
	"sync"

	"mlib.com/mcommu"
)

type MsgInfo struct {
	msgType reflect.Type
	handler func(conn mcommu.IConn, req interface{})
	id      uint32
}

const (
	header_len          = 8
	max_pkg_payload_len = 1024 - 8
)

type baseProcessor struct {
	msgInfos sync.Map
}

func (p *baseProcessor) HeaderLen() int {
	return header_len
}

// 4bytes  4bytes           nbytes
// cmd    len(8+msglen)      msg
// return leftlen, headid, payload, err
func (p *baseProcessor) ParseHeader(data []byte) (interface{}, int, error) {
	if data == nil || len(data) != p.HeaderLen() {
		return nil, 0, fmt.Errorf("invalid arg(%#v)", data)
	}
	// log.Printf("[D]ParseHeader:%#v\n", data)

	cmd := uint32(binary.BigEndian.Uint32(data[:4]))
	msglen := int(binary.BigEndian.Uint32(data[4:8]))
	// log.Printf("[D]Unmarshal cmd=0x%08x, msglen=%d\n", cmd, msglen)
	if msglen-p.HeaderLen() > max_pkg_payload_len {
		log.Printf("[E]the pkg payloadlen(%d) is too long, this is malicious connect, close it\n", msglen-header_len)
		return nil, 0, fmt.Errorf("the pkg payloadlen(%d) is too long, this is malicious connect, close it", msglen-header_len)
	}

	return cmd, msglen - p.HeaderLen(), nil
}

func (p *baseProcessor) RegisterHandler(headerid uint32, msg interface{}, handler func(conn mcommu.IConn, req interface{})) error {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Printf("[E]message pointer required\n")
		return fmt.Errorf("message pointer required")
	}
	msgID := msgType.Elem().Name()
	if msgID == "" {
		log.Printf("[E]unnamed json message\n")
		return fmt.Errorf("unnamed json message")
	}

	if _, ok := p.msgInfos.Load(msgID); ok {
		log.Printf("[E]msg(%s) is already registered\n", msgID)
		return fmt.Errorf("[E]msg(%s) is already registered", msgID)
	} else {
		i := new(MsgInfo)
		i.msgType = msgType
		i.handler = handler
		i.id = headerid
		p.msgInfos.Store(msgID, i)
		return nil
	}
}

func (p *baseProcessor) Route(headid, msg interface{}) (func(conn mcommu.IConn, req interface{}), error) {
	var foundinfo *MsgInfo
	p.msgInfos.Range(func(key, value interface{}) bool {
		if info, ok := value.(*MsgInfo); ok && info != nil {
			if info.id == headid {
				foundinfo = info
				return false
			}
		}
		return true
	})
	if foundinfo == nil {
		log.Printf("[E]headid(%x) not register\n", headid)
		return nil, fmt.Errorf("headid(%x) not register", headid)
	}
	return foundinfo.handler, nil
}
