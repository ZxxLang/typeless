package proto_test

import (
	"errors"
	"fmt"
	"github.com/gohub/typeless/proto"
	"reflect"
	"testing"
	"unsafe"
)

var Builtins []interface{}
var Funs []interface{}

func init() {
	var (
		e error
		i int
	)
	Builtins = []interface{}{
		e, i,
		nil, errors.New("error"),
		float32(1), float64(1),
		int8(1), int16(1), int32(1), int64(1), int(1),
		uint8(1), uint16(1), uint32(1), uint64(1), uint(1),
		complex64(1), complex128(1),
		uintptr(unsafe.Pointer(&Builtins)), unsafe.Pointer(&Builtins),
		true, byte('1'), "string", rune('世'),
		[1]string{"array string"}, []string{"slice string"},
		map[string]int{"map": 0}, make(chan int), make(chan int, 1),
		interface{}(&Builtins), func(i int) error { return nil }, proto.T{},
	}
	Funs = []interface{}{
		errors.New, fmt.Println, reflect.TypeOf, testing.AllocsPerRun,
		proto.Type, func() {}, func(i int) error { return nil },
	}
}

// 人工识别
func TestBuiltin(T *testing.T) {
	for _, k := range Builtins {
		s := proto.Type(k)
		if k != nil && s != reflect.TypeOf(k).String() {
			fmt.Println(s, "\t\t", reflect.TypeOf(k))
		}
	}
}

// 人工识别
func TestFunc(T *testing.T) {
	for _, k := range Funs {
		if proto.Type(k) != reflect.TypeOf(k).String() {
			fmt.Println(proto.Type(k), "\t\t", reflect.TypeOf(k).String())
		}
	}
	fmt.Println(proto.Type(struct {
		ddd proto.ProtoType
		A   string
		a   string
		B   int
	}{}), "\t\t", struct{ A string }{A: "struct"})
}
