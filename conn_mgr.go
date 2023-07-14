package mcommu

type IConn interface {
	Close()
	Write(data interface{}) error
	RemoteAddr() string
	LocalAddr() string
}

type IConnMgr interface {
	CloseConn(conn *tcpConn)
	PendingWriteNum() int
	Processor() IProcessor
	AyncHandle(conn IConn, msg interface{}, handler func(conn IConn, req interface{}))
}
