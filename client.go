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

type Client struct {
	conn        *conn
	seq         uint64
	seqLock     sync.Mutex
	closeOnce   sync.Once
	closed      bool
	respHeadBuf []byte
	readyCh     chan bool
}

func NewClient(addr string) (*Client, error) {
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
	cli := &Client{
		respHeadBuf: make([]byte, respHeadLen),
		readyCh:     make(chan bool, 1),
		conn:        conn,
	}
	cli.setFree()
	return cli, nil
}

func (client *Client) waitFree() {
	<-client.readyCh
}

func (client *Client) setFree() {
	client.readyCh <- true
}

func (client *Client) setBusy() {
	<-client.readyCh
}

func (client *Client) getSeq() uint64 {
	client.seqLock.Lock()
	seq := client.seq
	client.seq++
	client.seqLock.Unlock()
	return seq
}

func (client *Client) Close() error {
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

func (client *Client) call(requestLine string, args []*RequestArg) (resp interface{}, err error) {
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
		err = client.conn.responseStream(reqStreamArg.Data, reqStreamArg.TypeName)
		if err != nil {
			return
		}
	}
	resp, err, streamResponse = client.parseResp()
	return
}

type response struct {
	seq      uint64
	typeKind uint16
	typeName string
	data     []byte
}

func (client *Client) readRespLine() (resp *response, err error) {
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

func (client *Client) parseResp() (resp interface{}, err error, streamResponse bool) {
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

func (client *Client) genStream(typeName string) (interface{}, error) {
	r := client.conn.bufr
	w := client.conn.rwc
	switch typeName {
	case "stream":
		return &chunkReadWriteCloser{
			client: client,
			readWriter: &readWriter{
				Reader: &chunkReader{bufr: r},
				Writer: &chunkWriter{w: w},
			}}, nil
	case "istream":
		return &chunkReadCloser{
			client:      client,
			chunkReader: &chunkReader{bufr: r},
		}, nil
	case "ostream":
		return &chunkWriteCloser{
			client:      client,
			chunkWriter: &chunkWriter{w: w},
		}, nil
	default:
		return nil, errBadStreamType
	}
}

type chunkReadCloser struct {
	*chunkReader
	client *Client
}

func (crc *chunkReadCloser) Close() error {
	_, err := io.Copy(ioutil.Discard, crc.chunkReader)
	crc.client.setFree()
	return err
}

type chunkWriteCloser struct {
	*chunkWriter
	client *Client
}

func (cwc *chunkWriteCloser) Write(p []byte) (int, error) {
	//chunk编码中写入0\r\n\r\n代表结束
	//防止用户write一个空切片时误触发chunkWriter的结束
	if len(p) == 0 {
		return 0, nil
	}
	return cwc.chunkWriter.Write(p)
}

func (cwc *chunkWriteCloser) Close() error {
	_, err := cwc.chunkWriter.Write(nil)
	cwc.client.setFree()
	return err
}

type chunkReadWriteCloser struct {
	*readWriter
	client *Client
}

func (crwc *chunkReadWriteCloser) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return crwc.readWriter.Write(p)
}

func (crwc *chunkReadWriteCloser) Close() error {
	_, err := io.Copy(ioutil.Discard, crwc.readWriter)
	_, er := crwc.readWriter.Write(nil)
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
func (client *Client) Call(service, method string, args ...*RequestArg) (resp interface{}, err error) {
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
