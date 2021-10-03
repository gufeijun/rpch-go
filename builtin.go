package rpch

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

var builtinMarshal = map[reflect.Kind]func(v reflect.Value) []byte{
	reflect.Int8: int8Marshal, reflect.Int16: int16Marshal, reflect.Int32: int32Marshal, reflect.Int64: int64Marshal,
	reflect.Uint8: uint8Marshal, reflect.Uint16: uint16Marshal, reflect.Uint32: uint32Marshal, reflect.Uint64: uint64Marshal,
	reflect.Float32: float32Marshal, reflect.Float64: float64Marshal, reflect.String: stringMarshal, reflect.Bool: boolMarshal,
}

func put16(buf []byte, v uint16) {
	binary.LittleEndian.PutUint16(buf, v)
}

func put32(buf []byte, v uint32) {
	binary.LittleEndian.PutUint32(buf, v)
}

func put64(buf []byte, v uint64) {
	binary.LittleEndian.PutUint64(buf, v)
}

func get16(buf []byte) uint16 {
	return binary.LittleEndian.Uint16(buf)
}

func get32(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf)
}

func get64(buf []byte) uint64 {
	return binary.LittleEndian.Uint64(buf)
}

func _putHeader(typeKind uint16, name string, size int, call func([]byte)) []byte {
	buf := make([]byte, 8+len(name)+size)
	put16(buf[:2], typeKind)
	put16(buf[2:4], uint16(len(name)))
	put32(buf[4:8], uint32(size))
	copy(buf[8:8+len(name)], name)
	if call != nil {
		call(buf[8+len(name):])
	}
	return buf
}

func putHeader(name string, size int, call func([]byte)) []byte {
	return _putHeader(typeKind_Normal, name, size, call)
}

func int8Marshal(v reflect.Value) []byte {
	return putHeader("int8", 1, func(buf []byte) {
		buf[0] = byte(v.Interface().(int8))
	})
}

func int16Marshal(v reflect.Value) []byte {
	return putHeader("int16", 2, func(buf []byte) {
		put16(buf, uint16(v.Interface().(int16)))
	})
}

func int32Marshal(v reflect.Value) []byte {
	return putHeader("int32", 4, func(buf []byte) {
		put32(buf, uint32(v.Interface().(int32)))
	})
}

func int64Marshal(v reflect.Value) []byte {
	return putHeader("int64", 8, func(buf []byte) {
		put64(buf, uint64(v.Interface().(int64)))
	})
}

func uint8Marshal(v reflect.Value) []byte {
	return putHeader("uint8", 1, func(buf []byte) {
		buf[0] = v.Interface().(uint8)
	})
}

func uint16Marshal(v reflect.Value) []byte {
	return putHeader("uint16", 2, func(buf []byte) {
		put16(buf, v.Interface().(uint16))
	})
}

func uint32Marshal(v reflect.Value) []byte {
	return putHeader("uint32", 4, func(buf []byte) {
		put32(buf, v.Interface().(uint32))
	})
}

func uint64Marshal(v reflect.Value) []byte {
	return putHeader("uint64", 8, func(buf []byte) {
		put64(buf, v.Interface().(uint64))
	})
}

func float32Marshal(v reflect.Value) []byte {
	return putHeader("float32", 4, func(buf []byte) {
		put32(buf, uint32(v.Interface().(float32)))
	})
}

func float64Marshal(v reflect.Value) []byte {
	return putHeader("float64", 8, func(buf []byte) {
		put64(buf, uint64(v.Interface().(float64)))
	})
}

func stringMarshal(v reflect.Value) []byte {
	str := v.Interface().(string)
	return putHeader("string", len(str), func(buf []byte) {
		copy(buf, []byte(str))
	})
}

func boolMarshal(v reflect.Value) []byte {
	var ch byte = 0
	if v.Bool() {
		ch = 1
	}
	return putHeader("bool", 1, func(buf []byte) {
		buf[0] = ch
	})
}

// TODO delete this
func Unmarshal(typeName string, buf []byte) (*reflect.Value, error) {
	f, ok := builtinUnmarshal[typeName]
	if !ok {
		return nil, fmt.Errorf("no such func for %s", typeName)
	}
	return f(buf)
}

var builtinUnmarshal = map[string]func([]byte) (*reflect.Value, error){
	"int8": int8Unmarshal, "int16": int16Unmarshal, "int32": int32Unmarshal, "int64": int64Unmarshal,
	"uint8": uint8Unmarshal, "uint16": uint16Unmarshal, "uint32": uint32Unmarshal, "uint64": uint64Unmarshal,
	"float32": float32Unmarshal, "float64": float64Unmarshal, "string": stringUnmarshal, "bool": boolUnmarshal,
}

func genErr(expectLen int, _type string) error {
	return newProtoError(fmt.Sprintf("rpch: expect argument type: %s which expected to be %d bytes", _type, expectLen))
}

func newType(expectedLen int, name string, buf []byte, call func(buf []byte) *reflect.Value) (*reflect.Value, error) {
	if len(buf) < expectedLen {
		return nil, genErr(expectedLen, name)
	}
	return call(buf[:expectedLen]), nil
}

func boolUnmarshal(buf []byte) (*reflect.Value, error) {
	return newType(1, "bool", buf, func(buf []byte) *reflect.Value {
		num := buf[0] == 1
		v := reflect.ValueOf(num)
		return &v
	})
}

func uint8Unmarshal(buf []byte) (*reflect.Value, error) {
	return newType(1, "uint8", buf, func(buf []byte) *reflect.Value {
		v := reflect.ValueOf(uint8(buf[0]))
		return &v
	})
}

func int8Unmarshal(buf []byte) (*reflect.Value, error) {
	return newType(1, "int8", buf, func(buf []byte) *reflect.Value {
		v := reflect.ValueOf(int8(buf[0]))
		return &v
	})
}

func uint16Unmarshal(buf []byte) (*reflect.Value, error) {
	return newType(2, "uint16", buf, func(buf []byte) *reflect.Value {
		num := get16(buf)
		v := reflect.ValueOf(num)
		return &v
	})
}

func int16Unmarshal(buf []byte) (*reflect.Value, error) {
	return newType(2, "int16", buf, func(buf []byte) *reflect.Value {
		num := get16(buf)
		v := reflect.ValueOf(int16(num))
		return &v
	})
}

func uint32Unmarshal(buf []byte) (*reflect.Value, error) {
	return newType(4, "uint32", buf, func(buf []byte) *reflect.Value {
		num := get32(buf)
		v := reflect.ValueOf(num)
		return &v
	})
}

func int32Unmarshal(buf []byte) (*reflect.Value, error) {
	return newType(4, "int32", buf, func(buf []byte) *reflect.Value {
		num := get32(buf)
		v := reflect.ValueOf(int32(num))
		return &v
	})
}

func uint64Unmarshal(buf []byte) (*reflect.Value, error) {
	return newType(8, "uint64", buf, func(buf []byte) *reflect.Value {
		num := get64(buf)
		v := reflect.ValueOf(num)
		return &v
	})
}

func int64Unmarshal(buf []byte) (*reflect.Value, error) {
	return newType(8, "int64", buf, func(buf []byte) *reflect.Value {
		num := get64(buf)
		v := reflect.ValueOf(int64(num))
		return &v
	})
}

func float32Unmarshal(buf []byte) (*reflect.Value, error) {
	return newType(4, "float32", buf, func(buf []byte) *reflect.Value {
		num := get32(buf)
		v := reflect.ValueOf(float32(num))
		return &v
	})
}

func float64Unmarshal(buf []byte) (*reflect.Value, error) {
	return newType(8, "float64", buf, func(buf []byte) *reflect.Value {
		num := get64(buf)
		v := reflect.ValueOf(float64(num))
		return &v
	})
}

func stringUnmarshal(buf []byte) (*reflect.Value, error) {
	v := reflect.ValueOf(string(buf))
	return &v, nil
}

func IsBuiltinType(t string) bool {
	_, ok := builtinUnmarshal[t]
	return ok
}

func GetTypeKind(t string) uint16 {
	var tk uint16
	if IsBuiltinType(t) {
		tk = 0
	} else if t == "stream" || t == "istream" || t == "ostream" {
		tk = 1
	} else {
		tk = 2
	}
	return tk
}
