package main

import (
	"example/math/gfj"
	"fmt"
	"time"

	rpch "github.com/gufeijun/rpch-go"
)

type mathService struct{}

func (*mathService) Add(a uint32, b uint32) (uint32, error) {
	return a + b, nil
}

func (*mathService) Sub(a int32, b int32) (int32, error) {
	return a - b, nil
}

func (*mathService) Multiply(nums *gfj.TwoNum) (int32, error) {
	return nums.A * nums.B, nil
}

func (*mathService) Divide(a uint64, b uint64) (*gfj.Quotient, error) {
	quo := new(gfj.Quotient)
	quo.Quo = a / b
	quo.Rem = a % b
	return quo, nil
}

const addr = "127.0.0.1:8080"

func startServer() {
	svr := rpch.NewServer()
	gfj.RegisterMathService(new(mathService), svr)
	panic(svr.ListenAndServe("tcp", addr))
}

func main() {
	go startServer()
	time.Sleep(time.Second)
	conn, err := rpch.NewClient(addr)
	if err != nil {
		panic(err)
	}
	client := gfj.NewMathServiceClient(conn)
	{
		res, err := client.Add(1, 2)
		assert(err, res == 1+2, "Add failed")
	}
	{
		res, err := client.Divide(5, 2)
		assert(err, res.Quo == 5/2 && res.Rem == 5%2, "Divide failed")
	}
	{
		res, err := client.Multiply(&gfj.TwoNum{A: 2, B: 3})
		assert(err, res == 2*3, "Multiply failed")
	}
	{
		res, err := client.Sub(1, 100)
		assert(err, res == int32(1-100), "Sub failed")
	}
	fmt.Println("test success!")
}

func assert(err error, condition bool, msg string) {
	if err != nil {
		panic(err)
	}
	if !condition {
		panic(msg)
	}
}
