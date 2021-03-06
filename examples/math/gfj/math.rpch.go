// This is code generated by hgen. DO NOT EDIT!!!
// hgen version: v0.1.1
// source: math.gfj

package gfj

import (
	"encoding/json"
	rpch "github.com/gufeijun/rpch-go"
)

type Quotient struct {
	Quo uint64
	Rem uint64
}

type TwoNum struct {
	A int32
	B int32
}

type MathService interface {
	Add(uint32, uint32) (uint32, error)
	Sub(int32, int32) (int32, error)
	Multiply(*TwoNum) (int32, error)
	Divide(uint64, uint64) (*Quotient, error)
}

func RegisterMathService(impl MathService, svr *rpch.Server) {
	methods := map[string]*rpch.MethodDesc{
		"Add":      rpch.BuildMethodDesc(impl, "Add", "uint32"),
		"Sub":      rpch.BuildMethodDesc(impl, "Sub", "int32"),
		"Multiply": rpch.BuildMethodDesc(impl, "Multiply", "int32"),
		"Divide":   rpch.BuildMethodDesc(impl, "Divide", "Quotient"),
	}
	service := &rpch.Service{
		Impl:    impl,
		Name:    "Math",
		Methods: methods,
	}
	svr.Register(service)
}

func init() {
	rpch.RegisterMessage("Quotient", new(Quotient))
	rpch.RegisterMessage("TwoNum", new(TwoNum))
}

type MathServiceClient struct {
	conn *rpch.Conn
}

func NewMathServiceClient(conn *rpch.Conn) *MathServiceClient {
	return &MathServiceClient{
		conn: conn,
	}
}

func (c *MathServiceClient) Add(arg1 uint32, arg2 uint32) (res uint32, err error) {
	resp, err := c.conn.Call("Math", "Add",
		&rpch.RequestArg{
			TypeKind: 0,
			TypeName: "uint32",
			Data:     arg1,
		},
		&rpch.RequestArg{
			TypeKind: 0,
			TypeName: "uint32",
			Data:     arg2,
		})
	if resp == nil {
		return
	}
	return resp.(uint32), err
}

func (c *MathServiceClient) Sub(arg1 int32, arg2 int32) (res int32, err error) {
	resp, err := c.conn.Call("Math", "Sub",
		&rpch.RequestArg{
			TypeKind: 0,
			TypeName: "int32",
			Data:     arg1,
		},
		&rpch.RequestArg{
			TypeKind: 0,
			TypeName: "int32",
			Data:     arg2,
		})
	if resp == nil {
		return
	}
	return resp.(int32), err
}

func (c *MathServiceClient) Multiply(arg1 *TwoNum) (res int32, err error) {
	resp, err := c.conn.Call("Math", "Multiply",
		&rpch.RequestArg{
			TypeKind: 2,
			TypeName: "TwoNum",
			Data:     arg1,
		})
	if resp == nil {
		return
	}
	return resp.(int32), err
}

func (c *MathServiceClient) Divide(arg1 uint64, arg2 uint64) (res *Quotient, err error) {
	resp, err := c.conn.Call("Math", "Divide",
		&rpch.RequestArg{
			TypeKind: 0,
			TypeName: "uint64",
			Data:     arg1,
		},
		&rpch.RequestArg{
			TypeKind: 0,
			TypeName: "uint64",
			Data:     arg2,
		})
	if resp == nil {
		return
	}
	res = new(Quotient)
	return res, json.Unmarshal(resp.([]byte), res)

}
