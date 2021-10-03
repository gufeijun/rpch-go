package rpch

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"reflect"
)

const (
	typeKind_Normal = iota
	typeKind_Stream
	typeKind_Message
	typeKind_Error
)

const headLen = 8

type netArg struct {
	typeKind     uint32
	typeName     []byte
	dataLen      uint32
	typeNameLen  uint32
	data         []byte
	conn         *conn
	streamReader io.Reader
	streamWriter io.Writer
}

func (ra *netArg) unMarshal() (*reflect.Value, error) {
	switch ra.typeKind {
	case typeKind_Normal:
		return ra.builtinToGlangType()
	case typeKind_Stream:
		return ra.streamToGlangType()
	case typeKind_Message:
		return ra.messageToGlangType()
	default:
		return nil, errInvalidKind
	}
}

func (ra *netArg) builtinToGlangType() (*reflect.Value, error) {
	f, ok := builtinUnmarshal[string(ra.typeName)]
	if !ok {
		return nil, errBadRequestType
	}
	return f(ra.data)
}

func (ra *netArg) messageToGlangType() (*reflect.Value, error) {
	msg, ok := messageNameIDL2Golang[string(ra.typeName)]
	if !ok {
		return nil, errBadRequestMessage
	}
	value := reflect.New(reflect.TypeOf(msg))
	return &value, json.Unmarshal(ra.data, value.Interface())
}

type readWriter struct {
	io.Reader
	io.Writer
}

func (ra *netArg) streamToGlangType() (*reflect.Value, error) {
	var v reflect.Value
	switch string(ra.typeName) {
	case "stream":
		var rw io.ReadWriter = &readWriter{
			Reader: &chunkReader{bufr: ra.conn.bufr},
			Writer: &chunkWriter{w: ra.conn.rwc},
		}
		ra.streamReader = rw
		ra.streamWriter = rw
		v = reflect.ValueOf(rw)
	case "istream":
		var r io.Reader = &chunkReader{bufr: ra.conn.bufr}
		ra.streamReader = r
		v = reflect.ValueOf(r)
	case "ostream":
		var w io.Writer = &chunkWriter{w: ra.conn.rwc}
		ra.streamWriter = w
		v = reflect.ValueOf(w)
	default:
		return nil, errBadStreamType
	}
	return &v, nil
}

type netArgReader struct {
	conn    *conn
	curArg  *netArg
	headBuf []byte
}

func newNetArgReader(conn *conn) *netArgReader {
	return &netArgReader{
		conn:    conn,
		headBuf: make([]byte, headLen),
		curArg:  nil,
	}
}

// when encountering @headLen consecutive zero,
//it indicates that ther are no more args. we return io.EOF.
func (ar *netArgReader) nextArg() (*netArg, error) {
	if err := ar.readHeadBytes(); err != nil {
		return nil, err
	}
	arg := &netArg{
		typeKind:    uint32(binary.LittleEndian.Uint16(ar.headBuf[:2])),
		typeNameLen: uint32(binary.LittleEndian.Uint16(ar.headBuf[2:4])),
		dataLen:     uint32(binary.LittleEndian.Uint32(ar.headBuf[4:8])),
		conn:        ar.conn,
	}
	ar.curArg = arg
	if err := ar.readTypeName(); err != nil {
		return nil, err
	}
	arg.data = make([]byte, arg.dataLen)
	_, err := io.ReadFull(ar.conn.bufr, arg.data)
	return arg, err
}

func (ar *netArgReader) readTypeName() error {
	ar.curArg.typeName = make([]byte, int(ar.curArg.typeNameLen))
	_, err := io.ReadFull(ar.conn.bufr, ar.curArg.typeName)
	return err
}

func (ar *netArgReader) readHeadBytes() error {
	n, err := io.ReadFull(ar.conn.bufr, ar.headBuf)
	if err != nil {
		return err
	}
	if n < headLen {
		return errShortRead
	}
	return nil
}

type request struct {
	conn         *conn
	service      string
	method       string
	seq          uint64
	argCnt       uint32
	argReader    *netArgReader
	streamingArg *netArg
}

func (req *request) finishRequest() error {
	return req.conn.bufw.Flush()
}

func (req *request) parseArgs() (values []reflect.Value, err error) {
	var args []*netArg
	for i := 0; i < int(req.argCnt); i++ {
		arg, err := req.argReader.nextArg()
		if err != nil {
			return nil, err
		}
		if arg.typeKind == typeKind_Stream {
			req.streamingArg = arg
		}
		args = append(args, arg)
	}

	for i := 0; i < int(req.argCnt); i++ {
		value, err := args[i].unMarshal()
		if err != nil {
			return nil, err
		}
		values = append(values, *value)
	}
	return
}
