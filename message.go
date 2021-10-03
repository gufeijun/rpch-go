package rpch

import "reflect"

var messageNameIDL2Golang = make(map[string]interface{})

func isPtr(msg interface{}) bool {
	return reflect.ValueOf(msg).Type().Kind() == reflect.Ptr
}

func RegisterMessage(IDLName string, msg interface{}) {
	for isPtr(msg) {
		msg = reflect.Indirect(reflect.ValueOf(msg)).Interface()
	}
	messageNameIDL2Golang[IDLName] = msg
}
