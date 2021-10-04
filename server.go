package rpch

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"reflect"
	"sync"
	"time"
)

type Server struct {
	ReadTimeOut  time.Duration
	WriteTimeOut time.Duration
	services     sync.Map
}

var DefaultServer = NewServer()

func NewServer() *Server {
	return &Server{
		ReadTimeOut:  10 * time.Second,
		WriteTimeOut: 10 * time.Second,
	}
}

func (svr *Server) ListenAndServe(network string, addr string) error {
	l, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	return svr.Serve(l)
}

func (svr *Server) Serve(l net.Listener) error {
	var tempDelay time.Duration
	for {
		rwc, err := l.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0
		c := newConn(svr, rwc)
		go func() {
			err := svr.handleConn(c)
			if err != nil && err != io.EOF {
				log.Println(err)
			}
		}()
	}
}

func (svr *Server) handleConn(conn *conn) error {
	var err error
	var req *request
	defer func() {
		if e := recover(); e != nil {
			log.Printf("err recovered, err=%v\n", e)
		}
		conn.close()
	}()
	if err = conn.setReadDeadline(); err != nil {
		return err
	}
	_magic, err := conn.readMagic()
	if err != nil {
		return err
	}
	if magic != _magic {
		return errInvalidMagic
	}
	for {
		req, err = conn.readRequest()
		if err != nil {
			return err
		}
		if err = svr.handleRequest(req); err != nil {
			return err
		}
		if err = req.finishRequest(); err != nil {
			return err
		}
	}
}

func (svr *Server) handleRequest(req *request) error {
	iservice, ok := svr.services.Load(req.service)
	if !ok {
		return errBadRequestService
	}
	service := iservice.(*Service)
	methodDesc, ok := service.Methods[req.method]
	if !ok {
		return errBadRequestMethod
	}
	//NumIn还包括receiver这个参数，所以+1
	if methodDesc.MethodType.NumIn() != int(req.argCnt)+1 {
		return errBadRequestArgCnt
	}
	values, err := req.parseArgs()
	if err != nil {
		return err
	}
	rtns := methodDesc.Method.Call(values)
	if req.streamingArg != nil {
		//consume the rest data in istream if user doesn't do that in handler
		//otherwise it will affect the parse of the next request
		if r := req.streamingArg.streamReader; r != nil {
			io.Copy(ioutil.Discard, r)
		}
		// if stream is a ostream, we need to make w(chunkWriter) send an EOF signal to client after
		// handler, which indicates that there are no more data to be written to ostream.
		// Only by this, can client know it's time to accept Return Value of registered methods
		if w := req.streamingArg.streamWriter; w != nil {
			//it will send 0\r\n\r\n
			w.Write(nil)
		}
	}
	return req.conn.sendResponse(rtns, methodDesc, req.seq)
}

// a valid method should have at least one and at most two return values
// the last return value must impletement error interface which
// return the detail of the occured error
func checkServiceValidation(service *Service) {
	for _, methodDesc := range service.Methods {
		f := methodDesc.MethodType
		out := f.NumOut()
		lastType := f.Out(out - 1)
		ierror := reflect.TypeOf((*error)(nil)).Elem()
		if !lastType.Implements(ierror) || out != 1 && out != 2 {
			var err string
			//use goroutine, protect panic from getting recovered
			//this happens when developers edit and modify the code automatically generated by hgen
			if out == 1 || out == 2 {
				err = "last return value should implement interface error"
			} else {
				err = "Register Method should have at most 1 return values besides error"
			}
			go func() {
				panic(err)
			}()
		}
	}
}

func (svr *Server) Register(service *Service) {
	if service == nil {
		panic("register a nil service")
	}
	checkServiceValidation(service)
	svr.services.Store(service.Name, service)
}

func (svr *Server) UnRegister(serviceName string) {
	svr.services.Delete(serviceName)
}

func UnRegister(serviceName string) {
	DefaultServer.UnRegister(serviceName)
}

func Register(service *Service) {
	DefaultServer.Register(service)
}

func Serve(l net.Listener) error {
	return DefaultServer.Serve(l)
}

func ListenAndServe(network string, addr string) error {
	return DefaultServer.ListenAndServe(network, addr)
}

func Go(f func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered err: %v\n", err)
			}
		}()
		f()
	}()
}
