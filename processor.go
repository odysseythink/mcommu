package mcommu

type HandlerFunc func(conn IConn, req interface{})

// type IProcessor interface {
// 	// Route(msg interface{}) func(conn IConn, req interface{})
// 	// must goroutine safe
// 	ParsePkg(data []byte) (int, interface{}, []byte, error)
// 	UnmarshalPayload(payload []byte, msg interface{}) error
// 	// must goroutine safe
// 	Marshal(msg interface{}, info *MsgInfo) ([]byte, error)
// 	// RegisterHandler(cmd uint32, msg interface{}, handler func(conn IConn, req interface{})) error
// }

type IProcessor interface {

	// must goroutine safe,
	Marshal(msg interface{}) ([]byte, error)

	HeaderLen() int
	ParseHeader(data []byte) (msgid interface{}, msglen int, err error)
	// must goroutine safe
	Unmarshal(msgid interface{}, msgdata []byte) (msg interface{}, err error)
	// must goroutine safe
	Route(headid, msg interface{}) (func(conn IConn, req interface{}), error)
}
