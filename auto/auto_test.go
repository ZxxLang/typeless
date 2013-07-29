package auto_test

import (
	. "github.com/gohub/typeless/auto"
	"testing"
)

func init() {
	Conv.Register(func(a, b string) string { return a + b })
}

func TestInt(T *testing.T) {
	v, err := Conv.To(1, "10")
	if err != nil {
		T.Fatal(err)
	}
	want := 10
	if v.(int) != want {
		T.Errorf("want int %v but got &v", want, v)
	}

	v, err = Conv.To(int8(1), "10")
	if err != nil {
		T.Fatal(err)
	}
	want = 10
	if v.(int8) != int8(want) {
		T.Errorf("want int8 %v but got &v", want, v)
	}
}

func TestStringInt(T *testing.T) {
	v, err := Conv.To(1, "10", "1")
	if err != nil {
		T.Fatal(err)
	}
	want := 101
	if v.(int) != want {
		T.Errorf("want int %v but got &v", want, v)
	}
}

func TestError(T *testing.T) {
	_, err := Conv.To(1, "a0", "1")
	if err == nil {
		T.Error("want an error")
	}
}
