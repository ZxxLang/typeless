package caller_test

import (
	"fmt"
	. "github.com/gohub/typeless/caller"
	"strconv"
	"testing"
)

func TestZeroNumIn(T *testing.T) {
	err := New().Call(New).Error()
	if err != nil {
		T.Error(err)
	}
}

func TestFmtSprint(T *testing.T) {
	c := New().
		Call(fmt.Sprint, 1, true, "string")
	err := c.Error()
	if err != nil {
		T.Fatal(err)
	}
	out := c.Outs()
	if len(out) != 1 {
		T.Fatalf("expected out length 1, but %v", len(out))
	}
	o, s := fmt.Sprint(out[0]...), fmt.Sprint(1, true, "string")
	if o != s {
		T.Errorf("want: %s\n got: %s", s, o)
	}
}

func TestPush(T *testing.T) {
	c := New().
		Push(strconv.Itoa, 100)
	err := c.Error()
	if err != nil {
		T.Fatal(err)
	}
	out := c.Outs(0, 0)
	if len(out) != 1 {
		T.Fatalf("expected out length 1, but %v", len(out))
	}

	if len(out[0]) != 1 {
		T.Fatalf("expected out[0] length 1, but %v", len(out[0]))
	}

	if out[0][0].(string) != strconv.Itoa(100) {
		T.Errorf("want: %v\n got: %v", 100, out[0][0])
	}
}

func TestArgsPush(T *testing.T) {
	c := New().
		Push(100, strconv.Itoa)
	err := c.Error()
	if err != nil {
		T.Fatal(err)
	}
	out := c.Outs(0, 0)
	if len(out) != 1 {
		T.Fatalf("expected out length 1, but %v", len(out))
	}

	if len(out[0]) != 1 {
		T.Fatalf("expected out[0] length 1, but %v", len(out[0]))
	}

	if out[0][0].(string) != strconv.Itoa(100) {
		T.Errorf("want: %v\n got: %v", 100, out[0][0])
	}
}

func TestMore(T *testing.T) {
	c := New().
		Push(100, strconv.Itoa, func(s string) string { return s + "," + s })
	err := c.Error()
	if err != nil {
		T.Fatal(err)
	}
	out := c.Outs(0, 0)
	if len(out) != 2 {
		T.Fatalf("expected out length 2, but %v", len(out))
	}

	if len(out[1]) != 1 {
		T.Fatalf("expected out[1] length 1, but %v", len(out[1]))
	}

	if out[1][0].(string) != "100,100" {
		T.Errorf("want: %v\n got: %v", "100,100", out[1][0])
	}
}
func TestChainArgs(T *testing.T) {
	input := "123456789"
	v := uint64(0)
	err := New().
		Push(
		IsMinLen, input, 6,
		strconv.ParseUint, input, 10, 64).
		OutTo(&v)
	if err != nil || v != 123456789 {
		T.Fatalf("expected %v , but %v , error:", input, v, err)
	}
}

func IsMinLen(s string, min int) bool {
	min--
	for i, _ := range s {
		if i == min {
			return true
		}
	}
	return false
}

func IsRang(v, min, max uint64) bool {
	return v >= min && v <= max
}
