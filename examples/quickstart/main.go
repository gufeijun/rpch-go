package main

import (
	"example/quickstart/gfj"
	"fmt"
	"log"
	"time"

	rpch "github.com/gufeijun/rpch-go"
)

type mathService struct{}

func (*mathService) Add(a int32, b int32) (int32, error) {
	return a + b, nil
}

func startServer() {
	svr := rpch.NewServer()
	gfj.RegisterMathService(new(mathService), svr)
	panic(svr.ListenAndServe("tcp", "127.0.0.1:8080"))
}

func main() {
	go startServer()
	time.Sleep(time.Second)

	//客户端
	conn, err := rpch.NewClient("127.0.0.1:8080")
	if err != nil {
		panic(err)
	}
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
