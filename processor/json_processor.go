package processor

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
)

type JsonProcessor struct {
	baseProcessor
}

func (p *JsonProcessor) Unmarshal(data []byte) (interface{}, interface{}, int, error) {
	var payload []byte
	leftlen, headid, payload, err := p.ParsePkg(data)
	if err != nil {
		log.Printf("[E]ParsePkg failed:%v\n", err)
		return nil, nil, leftlen, err
	}
	var foundinfo *MsgInfo
	p.msgInfos.Range(func(key, value interface{}) bool {
		if info, ok := value.(*MsgInfo); ok && info != nil {
			if info.headerid == headid {
				foundinfo = info
				return false
			}
		}
		return true
	})
	if foundinfo == nil {
		log.Printf("[E]headid(%x) not register\n", headid)
		return nil, nil, leftlen, fmt.Errorf("headid(%x) not register", headid)
	}

	msg := reflect.New(foundinfo.msgType.Elem()).Interface()
	err = json.Unmarshal(payload, msg)
	if err != nil {
		log.Printf("[E]json.Unmarshal payload(%v) failed:%v\n", payload, err)
		return nil, nil, leftlen, fmt.Errorf("json.Unmarshal payload(%v) failed:%v", payload, err)
	}
	return headid, msg, leftlen, nil
}

// must goroutine safe
func (p *JsonProcessor) Marshal(msg interface{}) ([]byte, error) {
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
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("[E]json.Marshal payload failed:%v\n", err)
				return nil, fmt.Errorf("json.Marshal payload failed:%v", err)
			}

			ret := make([]byte, 8)
			binary.BigEndian.PutUint32(ret[:4], info.headerid)
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
