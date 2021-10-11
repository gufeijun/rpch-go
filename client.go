package rpch

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"reflect"
	"sync"
)

const respHeadLen = 16

type Conn struct {
	conn        *conn
	seq         uint64
	seqLock     sync.Mutex
	closeOnce   sync.Once
	closed      bool
	respHeadBuf []byte
	readyCh     chan bool
}

func Dial(addr string) (*Conn, error) {
	rwc, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 4)
	put32(buf, magic)
	if _, err = rwc.Write(buf); err != nil {
		return nil, err
	}
	conn := newConn(nil, rwc)
	cli := &Conn{
		respHeadBuf: make([]byte, respHeadLen),
		readyCh:     make(chan bool, 1),
		conn:        conn,
	}
	cli.setFree()
	return cli, nil
}

func (client *Conn) waitFree() {
	<-client.readyCh
}

func (client *Conn) setFree() {
	client.readyCh <- true
}

func (client *Conn) setBusy() {
	<-client.readyCh
}

func (client *Conn) getSeq() uint64 {
	client.seqLock.Lock()
	seq := client.seq
	client.seq++
	client.seqLock.Unlock()
	return seq
}

func (client *Conn) Close() error {
	var err error
	client.closeOnce.Do(func() {
		err = client.conn.rwc.Close()
	})
	return err
}

type RequestArg struct {
	TypeKind uint16
	TypeName string
	Data     interface{}
}

func (client *Conn) call(requestLine string, args []*RequestArg) (resp interface{}, err error) {
	client.waitFree()
	var streamResponse bool
	defer func() {
		if !streamResponse {
			client.setFree()
		}
	}()
	if _, err = io.WriteString(client.conn.bufw, requestLine); err != nil {
		return nil, err
	}
	var reqStreamArg *RequestArg
	for i := 0; i < len(args); i++ {
		if args[i].TypeKind == typeKind_Stream {
			if reqStreamArg != nil {
				return nil, errClientMultipleStream
			}
			reqStreamArg = args[i]
		}
		data, err := client.conn.marshal(reflect.ValueOf(args[i].Data), args[i].TypeKind, args[i].TypeName)
		if err != nil {
			return nil, err
		}
		if _, err := client.conn.bufw.Write(data); err != nil {
			return nil, err
		}
	}
	client.conn.bufw.Flush()
	if reqStreamArg != nil {
		err = client.sendStream(reqStreamArg)
		if err != nil {
			return
		}
	}
	resp, err, streamResponse = client.parseResp()
	return
}

func (client *Conn) sendStream(reqStreamArg *RequestArg) error {
	data := reqStreamArg.Data
	switch reqStreamArg.TypeName {
	case "istream":
		return client.conn.responseIStream(data.(io.Reader))
	case "ostream":
		return client.conn.responseOStream(data.(io.Writer))
	case "stream":
		return client.conn.responseIOStream(data.(io.ReadWriter))
	default:
		return errBadStreamType
	}
}

type response struct {
	seq      uint64
	typeKind uint16
	typeName string
	data     []byte
}

func (client *Conn) readRespLine() (resp *response, err error) {
	r := client.conn.bufr
	resp = new(response)
	buf := client.respHeadBuf
	if _, err = io.ReadFull(r, buf); err != nil {
		return
	}
	resp.seq = get64(buf[:8])
	resp.typeKind = get16(buf[8:10])
	typeNameLen := get16(buf[10:12])
	dataLen := get32(buf[12:16])
	buf = make([]byte, int(typeNameLen))
	if _, err = io.ReadFull(r, buf); err != nil {
		return
	}
	resp.typeName = string(buf)
	resp.data = make([]byte, dataLen)
	_, err = io.ReadFull(r, resp.data)
	return
}

func (client *Conn) parseResp() (resp interface{}, err error, streamResponse bool) {
	res, err := client.readRespLine()
	if err != nil {
		return
	}
	switch res.typeKind {
	case typeKind_Normal:
		f, ok := builtinUnmarshal[res.typeName]
		if !ok {
			err = errBadRequestType
			return
		}
		v, err := f(res.data)
		if err != nil {
			return nil, err, false
		}
		return (*v).Interface(), nil, false
	case typeKind_Error:
		return nil, &NonSeriousError{errMsg: string(res.data)}, false
	case typeKind_Message:
		return res.data, nil, false
	case typeKind_Stream:
		resp, err = client.genStream(res.typeName)
		streamResponse = true
		return
	case typeKind_NoRtnValue:
		return nil, nil, false
	default:
		return nil, errInvalidKind, false
	}
}

func (client *Conn) genStream(typeName string) (interface{}, error) {
	r := client.conn.bufr
	w := client.conn.rwc
	switch typeName {
	case "istream":
		fallthrough
	case "stream":
		return &chunkReadWriteCloser{
			client: client,
			readWriter: &readWriter{
				Reader: &chunkReader{bufr: r},
				Writer: &chunkWriter{w: w},
			}}, nil
	case "ostream":
		return &chunkWriteCloser{
			client:      client,
			chunkWriter: &chunkWriter{w: w},
		}, nil
	default:
		return nil, errBadStreamType
	}
}

type chunkWriteCloser struct {
	*chunkWriter
	client *Conn
	wLock  sync.Mutex
}

func (cwc *chunkWriteCloser) Write(p []byte) (int, error) {
	//chunk编码中写入0\r\n\r\n代表结束
	//防止用户write一个空切片时误触发chunkWriter的结束
	if len(p) == 0 {
		return 0, nil
	}
	cwc.wLock.Lock()
	n, err := cwc.chunkWriter.Write(p)
	cwc.wLock.Unlock()
	return n, err
}

func (cwc *chunkWriteCloser) Close() error {
	_, err := cwc.chunkWriter.Write(nil)
	cwc.client.setFree()
	return err
}

type chunkReadWriteCloser struct {
	rLock sync.Mutex
	wLock sync.Mutex
	*readWriter
	client *Conn
}

func (crwc *chunkReadWriteCloser) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	crwc.wLock.Lock()
	n, err := crwc.readWriter.Write(p)
	crwc.wLock.Unlock()
	return n, err
}

func (crwc *chunkReadWriteCloser) Read(p []byte) (int, error) {
	crwc.rLock.Lock()
	n, err := crwc.readWriter.Read(p)
	crwc.rLock.Unlock()
	return n, err
}

func (crwc *chunkReadWriteCloser) Close() error {
	crwc.wLock.Lock()
	_, er := crwc.readWriter.Write(nil)
	crwc.wLock.Unlock()
	_, err := io.Copy(ioutil.Discard, crwc)
	if err == nil {
		err = er
	}
	crwc.client.setFree()
	return err
}

type NonSeriousError struct {
	errMsg string
}

func (e *NonSeriousError) Error() string {
	return e.errMsg
}

func IsNonSeriousError(err error) bool {
	_, ok := err.(*NonSeriousError)
	return ok
}

//如果返回值是normal类型，则resp就是对应类型的value。
//如果是error类型，则resp就是nil，然后返回NonSeriousError
//如果是message类型，则resp是[]byte
//如果是stream类型，则resp就是io.ReadCloser、io.WriteCloser或者io.ReadWriteCloser
func (client *Conn) Call(service, method string, args ...*RequestArg) (resp interface{}, err error) {
	if client.closed {
		return nil, errClientClosed
	}
	defer func() {
		if e := recover(); e != nil {
			log.Printf("recovered err: %v\n", e)
		}
		if err != nil && !IsNonSeriousError(err) {
			client.closed = true
		}
	}()
	seq := client.getSeq()
	requestLine := fmt.Sprintf("%s %s %d %d\r\n", service, method, len(args), seq)
	return client.call(requestLine, args)
}
