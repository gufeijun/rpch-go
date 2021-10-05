package rpch

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"time"
)

const magic = 0x00686A6C
const seqSize = 8

type conn struct {
	svr       *Server
	rwc       net.Conn
	bufr      *bufio.Reader
	bufw      *errBufWriter
	closeOnce sync.Once
	seqsBuf   []byte
}

func newConn(svr *Server, rwc net.Conn) *conn {
	return &conn{
		seqsBuf: make([]byte, seqSize),
		svr:     svr,
		rwc:     rwc,
		bufr:    bufio.NewReader(rwc),
		bufw:    &errBufWriter{bufw: bufio.NewWriter(rwc)},
	}
}

type errBufWriter struct {
	bufw *bufio.Writer
	err  error
}

func (ew *errBufWriter) Write(buf []byte) (int, error) {
	if ew.err != nil {
		return 0, ew.err
	}
	n, err := ew.bufw.Write(buf)
	if err != nil {
		ew.err = err
		return 0, err
	}
	if n < len(buf) {
		ew.err = errShortWrite
	}
	return n, nil
}

func (ew *errBufWriter) Flush() error {
	if ew.err != nil {
		return ew.err
	}
	return ew.bufw.Flush()
}

func (c *conn) setReadDeadline() error {
	return c.rwc.SetReadDeadline(time.Now().Add(c.svr.ReadTimeOut))
}

func (c *conn) setWriteDeadline() error {
	return c.rwc.SetWriteDeadline(time.Now().Add(c.svr.WriteTimeOut))
}

func (c *conn) Read(buf []byte) (n int, err error) {
	return c.bufr.Read(buf)
}

func (c *conn) close() (err error) {
	c.closeOnce.Do(func() {
		err = c.rwc.Close()
	})
	return
}

func (c *conn) readMagic() (uint32, error) {
	buf := make([]byte, 4)
	n, err := c.Read(buf)
	if err != nil {
		return 0, err
	}
	if n < 4 {
		return 0, errShortRead
	}
	return binary.LittleEndian.Uint32(buf), nil
}

func (c *conn) readRequest() (req *request, err error) {
	line, err := readLine(c.bufr)
	if err != nil {
		return
	}
	req = new(request)
	if _, err = fmt.Sscanf(string(line), "%s%s%d%d", &req.service, &req.method, &req.argCnt, &req.seq); err != nil {
		return nil, errBadRequestLine
	}
	req.argReader = newNetArgReader(c)
	req.conn = c
	return
}

func (c *conn) sendError(err error) error {
	errMsg := err.Error()
	headBuf := _putHeader(typeKind_Error, "", len(errMsg), func(buf []byte) {
		copy(buf, []byte(errMsg))
	})
	c.bufw.Write(headBuf)
	return c.bufw.err
}

func (c *conn) sendNoRtnValue() error {
	headBuf := _putHeader(typeKind_NoRtnValue, "", 0, nil)
	c.bufw.Write(headBuf)
	return c.bufw.err
}

func (c *conn) sendResponse(rtns []reflect.Value, methodDesc *MethodDesc, seq uint64) error {
	if methodDesc.RetTypeKind == typeKind_Stream {
		//如果是返回stream的话，会有三个参数：stream，func()以及error
		if !rtns[1].IsNil() {
			defer rtns[1].Interface().(func())()
		}
	}
	put64(c.seqsBuf, seq)
	c.bufw.Write(c.seqsBuf)
	e := rtns[len(rtns)-1]
	if err := e.Interface(); err != nil {
		return c.sendError(err.(error))
	}
	if len(rtns) == 1 {
		return c.sendNoRtnValue()
	}
	buf, err := c.marshal(rtns[0], methodDesc.RetTypeKind, methodDesc.RetTypeName)
	if err != nil {
		return err
	}
	c.bufw.Write(buf)
	if c.bufw.err == nil && methodDesc.RetTypeKind == typeKind_Stream {
		c.bufw.err = c.responseStream(rtns[0].Interface(), methodDesc.RetTypeName)
	}
	return c.bufw.err
}

func (c *conn) marshal(v reflect.Value, typeKind uint16, typeName string) ([]byte, error) {
	if typeKind == typeKind_Normal {
		return builtinMarshal[v.Kind()](v), nil
	}
	if v.IsNil() {
		return nil, errBadResponse
	}
	switch typeKind {
	case typeKind_Message:
		data, err := json.Marshal(v.Interface())
		if err != nil {
			return nil, err
		}
		buf := _putHeader(typeKind_Message, typeName, len(data), func(b []byte) {
			copy(b, data)
		})
		return buf, nil
	case typeKind_Stream:
		buf := _putHeader(typeKind_Stream, typeName, 0, nil)
		return buf, nil
	}
	return nil, errInvalidKind
}

func (c *conn) responseStream(v interface{}, typeName string) error {
	c.bufw.Flush()
	switch typeName {
	case "istream":
		return c.responseIStream(v.(io.Reader))
	case "ostream":
		return c.responseOStream(v.(io.Writer))
	case "stream":
		return c.responseIOStream(v.(io.ReadWriter))
	default:
		return errBadStreamType
	}
}

func (c *conn) responseIOStream(rw io.ReadWriter) error {
	ch := make(chan error)
	Go(func() {
		ch <- c.responseOStream(rw)
	})
	err := c.responseIStream(rw)
	if err != nil {
		c.close()
	}
	er := <-ch
	if err == nil {
		err = er
	}
	return err
}

func (c *conn) responseOStream(w io.Writer) error {
	cr := &chunkReader{bufr: c.bufr}
	_, err := io.Copy(w, cr)
	return err
}

func (c *conn) responseIStream(r io.Reader) error {
	cw := &errWriter{w: &chunkWriter{w: c.rwc}}
	_, err := io.Copy(cw, r)
	cw.err = err
	cw.Write(nil)
	return cw.err
}

type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) Write(p []byte) (int, error) {
	if ew.err != nil {
		return 0, ew.err
	}
	var n int
	n, ew.err = ew.w.Write(p)
	return n, ew.err
}
