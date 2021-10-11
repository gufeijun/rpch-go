package local

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"

	"proxy/gfj"

	"github.com/gufeijun/rpch-go"
)

func StartLocalProxy(lisAddr string, remoteAddr string) error {
	listener, err := net.Listen("tcp", lisAddr)
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go func() {
			defer conn.Close()
			if err := handleConn(conn, remoteAddr); err != nil {
				panic(err)
			}
		}()
	}
}

func handleConn(localConn net.Conn, remoteAddr string) error {
	buf := make([]byte, 256)

	_, err := localConn.Read(buf)
	if err != nil || buf[0] != 0x05 {
		return errors.New("unsupported socks version")
	}

	localConn.Write([]byte{0x05, 0x00})

	n, err := localConn.Read(buf)
	if err != nil || n < 7 {
		return errors.New("invalid host addr")
	}

	if buf[1] != 0x01 {
		return errors.New("unsupported socks5 method")
	}

	var ip []byte
	switch buf[3] {
	case 0x01:
		ip = buf[4 : 4+net.IPv4len]
	case 0x03:
		ipAddr, err := net.ResolveIPAddr("ip", string(buf[5:n-2]))
		if err != nil {
			return err
		}
		ip = ipAddr.IP
	case 0x04:
		ip = buf[4 : 4+net.IPv6len]
	default:
		return errors.New("unknown host type")
	}
	port := binary.BigEndian.Uint16(buf[n-2:])
	dstAddr := &net.TCPAddr{
		IP:   ip,
		Port: int(port),
	}

	rpcConn, err := rpch.Dial(remoteAddr)
	if err != nil {
		return err
	}
	defer rpcConn.Close()
	client := gfj.NewDialServiceClient(rpcConn)

	dstConn, err := client.Dial(dstAddr.String())
	if err != nil {
		return err
	}
	defer dstConn.Close()

	localConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	go func() {
		_, err := io.Copy(localConn, dstConn)
		if err != nil {
			localConn.Close()
			dstConn.Close()
		}
	}()

	_, err = io.Copy(dstConn, localConn)
	return err
}
