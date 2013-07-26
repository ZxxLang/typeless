package auto_test

import (
	"fmt"
	. "github.com/gohub/typeless/auto"
	"testing"
)

func init() {
	Conv.Register(func(a, b string) string { return a + b })
}

func TestInt(T *testing.T) {
	v, err := Conv.To(1, "10")
	if err != nil || v.(int) != 10 {
		T.Fail()
	}
	v, err = Conv.To(int8(1), "10")
	if err != nil || v.(int8) != 10 {
		T.Fail()
	}
}

func TestStringInt(T *testing.T) {
	v, err := Conv.To(1, "10", "1")
	if err != nil || v.(int) != 101 {
		T.Fail()
	}
}

func TestError(T *testing.T) {
	_, err := Conv.To(1, "a0", "1")
	if err == nil {
		T.Fail()
	} else {
		fmt.Println(err)
	}
}
