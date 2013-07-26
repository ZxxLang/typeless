/*
proto 是一个关于对象原型处理的工具.采用完整 PkgPath 对类型进行描述.
比如 net/http 包中的

  type HandlerFunc func(ResponseWriter, *Request)

通常代码这样写

  func Handler(w http.ResponseWriter, r *http.Request) {
  }

类型描述为

  func(http.ResponseWriter, *http.Request)

这种描述省略了 http 包的 PkgPath, proto 解析出包含 PkgPath 的描述

  func(net/http.ResponseWriter, *net/http.Request)

这就是 proto.
为了简便, 其他与 reflect 不同的地方
  * error 直接描述为 error, 而非 *errors.errorString
  * interface{} 直接描述为 interface{}, 而非 interface {}
  * 匿名 struct ,直接返回完全字符串描述, 例如 `struct { A string; a string; B int }`
  * 特殊值 nil 直接描述为 nil, 而非 <nil>
*/
package proto

import (
	"reflect"
	"strconv"
	"strings"
)

// 返回 proto 描述
func Type(x interface{}) string {
	return prototype(x)
}

// 返回多个 interface{} 的 proto []string
func Types(in ...interface{}) (ss []string) {
	ss = make([]string, len(in))
	for i, v := range in {
		ss[i] = prototype(v)
	}
	return
}

// 获取 TypeOf
func TypeOf(x interface{}) reflect.Type {
	t, ok := x.(reflect.Type)
	if !ok {
		t = reflect.TypeOf(x)
	}
	return t
}

// 剥去 Ptr
func TypeIndirect(x interface{}) reflect.Type {
	t := TypeOf(x)
	if t.Kind() != reflect.Ptr {
		return t
	}
	return t.Elem()
}

// 获取 ValueOf
func ValueOf(i interface{}) (v reflect.Value) {
	switch x := i.(type) {
	case reflect.Value:
		return x
	case reflect.Type:
		return
	}
	return reflect.ValueOf(i)
}

// 剥去 Ptr
func ValueIndirect(i interface{}) (v reflect.Value) {
	v = ValueOf(i)
	if v.IsValid() {
		return reflect.Indirect(v)
	}
	return
}

func prototype(x interface{}) string {
	if x == nil {
		return "nil"
	}
	if _, ok := x.(error); ok {
		return "error"
	}

	t, ok := x.(reflect.Type)
	if !ok {
		t = reflect.TypeOf(x)
	}
	k := t.Kind()
	switch k {
	case reflect.Array:
		return "[" + strconv.Itoa(t.Len()) + "]" + prototype(t.Elem())
	case reflect.Chan:
		return t.ChanDir().String() + " " + prototype(t.Elem())
	case reflect.Func:
		s := "func("
		max := t.NumIn() - 1
		for i := 0; i <= max; i++ {
			if i != max {
				s += prototype(t.In(i)) + ", "
			} else {
				if t.IsVariadic() {
					s += "..." + prototype(t.In(i).Elem())
				} else {
					s += prototype(t.In(i))
				}
			}
		}
		max = t.NumOut() - 1

		if max > 0 {
			s += ") ("
		} else if max == 0 {
			s += ") "
		} else {
			s += ")"
		}

		for i := 0; i <= max; i++ {
			s += prototype(t.Out(i))
			if i != max {
				s += ", "
			}
		}
		if max > 0 {
			s += ")"
		}
		return s
	case reflect.Map:
		return "map[" + prototype(t.Key()) + "]" + prototype(t.Elem())
	case reflect.Ptr:
		return "*" + prototype(t.Elem())
	case reflect.Slice:
		return "[]" + prototype(t.Elem())
	case reflect.Interface:
		if t.Name() == "" {
			return "interface{}"
		}
		if t.PkgPath() == "" {
			return t.Name()
		}

		return t.PkgPath() + "." + t.Name()
	case reflect.Uintptr, reflect.UnsafePointer:
		return k.String()
	case reflect.Struct:
		p := t.PkgPath()
		n := t.Name()
		if p == "" {
			if n == "" {
				n = "struct { "
				for i := 0; i < t.NumField(); i++ {
					f := t.Field(i)
					if i == 0 {
						n += f.Name
					} else {
						n += "; " + f.Name
					}
					n += " " + prototype(f.Type)
				}
			}
			return n + " }"
		}
		return p + "." + n
	}
	pp := t.PkgPath()
	if pp == "" {
		return t.String()
	}
	return t.PkgPath() + "." + t.String()
}

// 判断是否是一个 reflect.Type
func IsType(i interface{}) bool {
	_, ok := i.(reflect.Type)
	return ok
}

// 判断是否是一个 reflect.Type
func IsValue(i interface{}) bool {
	_, ok := i.(reflect.Value)
	return ok
}

// 把根据参数来生成 proto 描述,并用 ", " 连接
func Join(in ...interface{}) string {
	max := len(in) - 1
	s := ""
	for i, v := range in {
		if i != max {
			s += prototype(v) + ", "
		} else {
			s += prototype(v)
		}
	}
	return s
}

// 如果 prefix 非空, 返回 "prefix, "+Join(in ...)
func JoinPrefix(prefix string, in ...interface{}) string {
	s := Join(in...)
	if s != "" && prefix != "" {
		return prefix + ", " + s
	}
	return prefix + s
}

// 便捷工具
func Slice(in ...interface{}) []interface{} {
	return in
}

// 由 In ,Out 两个数组生成 func 描述,算法很简单，提供此函数是为了保证结果符合 proto 的习惯.
func Func(in, out []interface{}) string {
	i := Join(in...)
	o := Join(out...)
	if i == "" {
		return i
	}
	i = "func(" + i + ")"
	l := len(out)
	if l == 1 {
		i += " " + o
	} else if l > 1 {
		i += " (" + o + ")"
	}
	return i
}

// 函数的 proto 描述，并把返回值分离开
// 返回值举例
//   fun : {"func(", "int", "string", "...interface{}", ")"}
//   out : {"(", "int", "error", ")"}
func FuncSplit(fn interface{}) (fun, out []string) {
	if fn == nil {
		return
	}
	t, ok := fn.(reflect.Type)
	if !ok {
		t = reflect.TypeOf(fn)
	}
	if t.Kind() != reflect.Func {
		return
	}
	fun = append(fun, "func(")
	max := t.NumIn() - 1
	for i := 0; i <= max; i++ {
		if t.IsVariadic() {
			fun = append(fun, "..."+prototype(t.In(i).Elem()))
		} else {
			fun = append(fun, prototype(t.In(i)))
		}
	}
	fun = append(fun, ")")
	max = t.NumOut() - 1

	if max > 0 {
		out = append(out, "(")
	}

	for i := 0; i <= max; i++ {
		out = append(out, prototype(t.Out(i)))
	}
	if max > 0 {
		out = append(out, ")")
	}
	return
}

type ProtoType interface {
	String() string
	Code(...string) string
	New(...interface{}) interface{}
}

// 类型描述
type T struct {
	Name, Type, Proto string
}

func (t *T) String() string {
	if t.Type == "" {
		return t.Proto
	}
	if t.Name == "" {
		return t.Type
	}
	if t.Proto == "" {
		return t.Name + " " + t.Type
	}
	return "var " + t.Name + " " + t.Type
}

func (t *T) Code(name ...string) string {
	if t.Type == "" {
		return ""
	}
	if len(name) != 0 {
		return "var " + strings.Join(name, ", ") + " " + t.Type
	}
	if t.Name == "" {
		return ""
	}
	return "var " + t.Name + " " + t.Type
}

// 实例描述
type Instance struct {
	T
	Value interface{}
}

// 函数
type Fn struct {
	T
	In, Out []T
}

// 接口
type Interface map[string]ProtoType

// 结构体
type Struct struct {
	Fields  map[string]T
	Methods map[string]Fn
}
