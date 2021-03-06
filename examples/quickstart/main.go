package main

import (
	"fmt"
	"log"
	"time"

	"quickstart/gfj"

	rpch "github.com/gufeijun/rpch-go"
)

const addr = "127.0.0.1:8080"

type mathService struct{}

func (*mathService) Add(a int32, b int32) (int32, error) {
	return a + b, nil
}

func startServer() {
	svr := rpch.NewServer()
	gfj.RegisterMathService(new(mathService), svr)
	panic(svr.ListenAndServe(addr))
}

func main() {
	go startServer()
	time.Sleep(time.Second)

	//客户端
	conn, err := rpch.Dial(addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := gfj.NewMathServiceClient(conn)
	result, err := client.Add(2, 3)
	if err != nil {
		panic(err)
	}
	if result != 5 {
		log.Panicf("want %d, but got %d\n", 5, result)
	}
	fmt.Println("test success!")
}
