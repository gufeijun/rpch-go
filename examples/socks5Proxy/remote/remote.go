package remote

import (
	"io"
	"net"

	"proxy/gfj"

	"github.com/gufeijun/rpch-go"
)

type dialService struct {
	addrCh chan string
}

func (ds *dialService) Dial(url string) (rw io.ReadWriter, onfinsh func(), err error) {
	dstAddr, err := net.ResolveTCPAddr("tcp", url)
	if err != nil {
		return nil, nil, err
	}
	dstConn, err := net.DialTCP("tcp", nil, dstAddr)
	if err != nil {
		return nil, nil, err
	}
	go func() {
		ds.addrCh <- dstConn.LocalAddr().String()
	}()
	return dstConn, func() {
		dstConn.Close()
	}, nil
}

func StartRemoteProxy(addr string, ch chan string) {
	svr := rpch.NewServer()
	gfj.RegisterDialService(&dialService{
		addrCh: ch,
	}, svr)
	panic(svr.ListenAndServe(addr))
}
