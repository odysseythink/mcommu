package processor

import (
	"encoding/binary"
	"fmt"
	"log"
	"reflect"

	"github.com/golang/protobuf/proto"
)

type ProtobufProcessor struct {
	baseProcessor
	// msgInfos map[string]*MsgInfo
	// msgInfos sync.Map
}

func (p *ProtobufProcessor) UnmarshalPayload(payload []byte, msg interface{}) error {
	if msg == nil {
		log.Printf("[W]invalid arg")
		return nil
	}
	if pbmsg, ok := msg.(proto.Message); !ok {
		log.Printf("[W]msg must be proto.Message type")
		return nil
	} else {
		return proto.Unmarshal(payload, pbmsg)
	}
}

func (p *ProtobufProcessor) Unmarshal(msgid interface{}, msgdata []byte) (interface{}, error) {
	var foundinfo *MsgInfo
	p.msgInfos.Range(func(key, value interface{}) bool {
		if info, ok := value.(*MsgInfo); ok && info != nil {
			if info.id == msgid {
				foundinfo = info
				return false
			}
		}
		return true
	})
	if foundinfo == nil {
		log.Printf("[E]headid(%x) not register\n", msgid)
		return nil, fmt.Errorf("headid(%x) not register", msgid)
	}

	msg := reflect.New(foundinfo.msgType.Elem()).Interface()
	if pbmsg, ok := msg.(proto.Message); !ok {
		log.Printf("[W]msg must be proto.Message type")
		return nil, fmt.Errorf("msg(%s) must be proto.Message type", foundinfo.msgType.Elem().Name())
	} else {
		err := proto.Unmarshal(msgdata, pbmsg)
		if err != nil {
			log.Printf("[W]proto.Unmarshal failed:%v\n", err)
			return nil, fmt.Errorf("proto.Unmarshal failed:%v", err)
		}
		return msg, nil
	}
}

// must goroutine safe
func (p *ProtobufProcessor) Marshal(msg interface{}) ([]byte, error) {
	if msg == nil {
		log.Printf("[W]invalid arg\n")
		return nil, fmt.Errorf("invalid arg")
	}
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Printf("[E]json message pointer required\n")
		return nil, fmt.Errorf("json message pointer required")
	}
	msgID := msgType.Elem().Name()
	if val, ok := p.msgInfos.Load(msgID); !ok {
		log.Printf("[E]message %v not registered\n", msgID)
		return nil, fmt.Errorf("message %v not registered", msgID)
	} else {
		if info, ok := val.(*MsgInfo); !ok && info == nil {
			log.Printf("[E]message %v not registered correct\n", msgID)
			return nil, fmt.Errorf("message %v not registered correct", msgID)
		} else {
			if _, ok := msg.(proto.Message); !ok {
				log.Printf("[W]msg(%s) must be proto.Message type\n", msgID)
				return nil, fmt.Errorf("msg must be proto.Message type")
			}
			data, err := proto.Marshal(msg.(proto.Message))
			if err != nil {
				log.Printf("[E]proto.Marshal payload failed:%v\n", err)
				return nil, fmt.Errorf("proto.Marshal payload failed:%v", err)
			}

			ret := make([]byte, 8)
			binary.BigEndian.PutUint32(ret[:4], info.id)
			binary.BigEndian.PutUint32(ret[4:8], uint32(8+len(data)))
			ret = append(ret, data...)
			// var hexbuilder strings.Builder
			// for _, v := range ret {
			// 	hexbuilder.WriteString(fmt.Sprintf("%02x ", v))
			// }
			// log.Printf("[D]send :%s\n", hexbuilder.String())
			return ret, nil
		}
	}
}
