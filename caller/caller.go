/*
caller 提供了一种函数调用链式操作, 发生 panic ,被调用函数最后一个值 false/error 终止调用链.
*/
package caller

import (
	"errors"
	"fmt"
	"reflect"
)

// Caller 接口
type Caller interface {
	Call(...interface{}) Caller
	Push(...interface{}) Caller
	Ok() bool
	Out(...int) [][]interface{}
	Error() error
}

// Caller 默认类型实现
type call struct {
	args   []interface{}
	outs   [][]interface{}
	failed bool
	err    error
}

// 返回 Call 实例接口对象
func New() Caller {
	c := new(call)
	c.args = []interface{}{}
	c.outs = [][]interface{}{}
	return c
}

// Call 方法,接收任意类型对象, 以参数是否是函数进行分类
// 函数之前的对象将按顺序压入备用参数切片, 遇到函数尝试执行函数,
// 函数执行的参数由备用参数和函数后续参数组成, 后续参数不够用时才使用备用参数.
// 举例
//   New().Call(1,2,"3",strconv.ParseInt,10,64,"more")
// 调用函数
//   strconv.ParseInt("3",10,64)
// 执行后备用参数包含
//   []interface{}{1,2,"more"}
func (p *call) Call(i ...interface{}) Caller {
	return p.call(false, i)
}

// Push 方法在 Call 方法的基础上, 把执行的返回值除了过滤类型值外压入备用参数切片.
// 过滤类型值是指, 返回值的最后一个值是 bool 或 error 类型.
// 举例
//   New().Call(1,2,"3",strconv.ParseInt,10,64,"more")
// 调用函数
//   strconv.ParseInt("3",10,64)
// 执行后备用参数包含
//   []interface{}{1,2,int64(3),"more"}
// 如果发生 panic , 忽略压入返回值
func (p *call) Push(i ...interface{}) Caller {
	return p.call(true, i)
}

func (p *call) call(push bool, is []interface{}) (caller Caller) {
	defer func() {
		if err := recover(); err != nil {
			p.err = errors.New(fmt.Sprint(err))
		}
	}()
	caller = p
	if p.failed || p.err != nil || len(is) == 0 {
		return
	}
	var fn reflect.Value
	var typ reflect.Type
	l := len(is)
	for i := 0; i < l; {
		fn = reflect.ValueOf(is[i])
		if reflect.Func != fn.Kind() {
			p.args = append(p.args, is[i])
			i++
			continue
		}
		typ = fn.Type()
		la := typ.NumIn()
		numIn := 0
		if typ.IsVariadic() || la != 0 {
			for j := 0; typ.IsVariadic() || j < la; j++ {
				i++
				if i >= l || reflect.Func == reflect.TypeOf(is[i]).Kind() {
					break
				}
				p.args = append(p.args, is[i])
				numIn++
			}
		} else if la == 0 {
			i++
		}

		la = len(p.args)
		if la < typ.NumIn() {
			p.setFailed(NotEnough)
			break
		}
		if !typ.IsVariadic() {
			numIn = typ.NumIn()
		}

		args := p.args[la-numIn:]
		p.args = p.args[:la-numIn]

		in := make([]reflect.Value, len(args))
		for i, v := range args {
			in[i] = reflect.ValueOf(v)
		}

		out := fn.Call(in)
		la = len(out)

		// 总是保存输出
		o := make([]interface{}, la)
		for i, v := range out {
			o[i] = v.Interface()
		}
		p.outs = append(p.outs, o)
		if la == 0 {
			continue
		}

		fn = out[la-1]
		switch fn.Kind().String() {
		case "bool":
			if !fn.Bool() {
				p.setFailed(Failed)
				return
			}
			la--
		case "error":
			if !fn.IsNil() {
				p.setFailed(fn.Interface().(error))
				return
			}
			la--
		}
		// 输出压入备用参数
		if !push {
			continue
		}

		for i := 0; i < la; i++ {
			p.args = append(p.args, out[i].Interface())
		}
	}
	return
}
func (p *call) setFailed(err error) {
	if err == nil {
		p.failed = true
	} else {
		p.err = err
	}
}

// 返回函数执行的结果, 内部保存形式是
//   [][]interface{}
// 对应执行的全部结果, 即便被调用函数没有返回值,也会产生一个
//   []interface{}
// 默认返回最后一个
// 参数如果是两个相等的数表示返回所有,否则表示返回一个切片
func (p *call) Out(i ...int) [][]interface{} {
	l := len(p.outs)

	s, e := 0, l
	if e > 1 {
		s = e - 1
	}
	ll := len(i)
	if ll >= 1 {
		s = i[0]
	}
	if ll > 1 {
		e = i[1]
	}
	if s < 0 {
		s = l + s
	}
	if e < 0 {
		e = l + e
	}
	if s > e {
		s, e = e, s
	}
	if s == e {
		return p.outs[:]
	}
	return p.outs[s:e]
}

// 以 bool 形式表示调用链过程中没有发生 panic/false/error
func (p *call) Ok() bool {
	return p.err == nil && !p.failed
}

// 表示如果调用链中产生的是 false
var Failed error = errors.New("call Failed")

// 表示参数数量不够
var NotEnough error = errors.New("call Not enough arguments")

// 以 error 形式表示调用链过程中发生 panic/false/error

func (p *call) Error() error {
	if p.err != nil {
		return p.err
	}
	if p.failed {
		return Failed
	}
	return p.err
}
