package rpch

import "reflect"

type MethodDesc struct {
	Method      reflect.Value
	RetTypeName string
	MethodType  reflect.Type
	RetTypeKind uint16
}

type Service struct {
	//the interface who implement all the methods of this service
	Impl    interface{}
	Methods map[string]*MethodDesc
	Name    string
}

func BuildMethodDesc(v interface{}, method string, retTypeName string) *MethodDesc {
	vv := reflect.ValueOf(v)
	tt, _ := vv.Type().MethodByName(method)
	return &MethodDesc{
		Method:      vv.MethodByName(method),
		MethodType:  tt.Func.Type(),
		RetTypeName: retTypeName,
		RetTypeKind: GetTypeKind(retTypeName),
	}
}
