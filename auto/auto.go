/*
auto 是一个类型自动执行器,试图在最大限度内进行类型的执行,除了内置的一些常规类型支持
还提供了一种注册机制,可以注册执行函数,auto 尝试自动匹配注册函数进行执行
*/
package auto

import (
	"errors"
	"fmt"
	"github.com/gohub/typeless/proto"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func toFail(s ...interface{}) error {
	return errors.New("Auto failed: " + fmt.Sprint(s...))
}
func toInValidArgs(s ...interface{}) error {
	return errors.New("Auto invalid arguments: " + fmt.Sprint(s...))
}
func toNotSupported(s ...interface{}) error {
	return errors.New("Auto not supported: " + fmt.Sprint(s...))
}

var (
	Conv = Group{} // 内置的执行器映射变量
)

// 对执行函数进行包装, 这是一个内部使用的结构, 暂未提供接口, 目前只是为了文档.
// 执行函数合法形式为, 不支持可变参数的函数
//   func(in Type0 [,arg1[,arg2]])(out Type[,ok bool/error])
// 返回值数量如果大于1, 那么最后一个返回值如果是 bool/error, 表示执行是否成功
type Fn struct {
	ify    string   //最后一个是否作为判断依据,并记录类型
	numout int      //除了最后一个判断依据外的有效输出个数
	used   bool     //是否被其他Fn使用
	name   string   //给函数命名
	args   []string //所有参数类型
	outs   []string //所有返回值类型
	queue  []string //链式,对应其他的key
	apply  reflect.Value
}

// 如果想为 Fn 定义一个名字, 通过下面的形式
//   FuncMap{"funcname":func(){}}
// 调用的时候也必须给出名字和参数，通过下面的形式
//   Call{"funcname",a1,a2,...}}
type FuncMap map[string]interface{}

// 配合 FuncMap 使用, 直接匹配到指定的函数
//   Call{"funcname",a1,a2,...}}
// 如果参数数量和类型与 funcname 完全一致, 表示忽略上一个的结果
type Call []interface{}

// 如果想快速的匹配到合适的函数, 通过下面的形式传入参数
//   Args{a1,a2,...}
type Args []interface{}

// 完整参数匹配, 忽略上一个函数的结果
type ArgsFull []interface{}

// 一组执行函数组成 Group, 到达自动匹配执行的效果
// 相同参数并且 out 相同,只能注册一个
// 内部使用了 map 类型, 并使用了 sync 锁.
type Group struct {
	All       []string //已经注册的执行器描述
	npcmatch  int
	lock      *sync.RWMutex
	lockclose *sync.RWMutex
	m         map[string]*Fn
	closeset  []string //已经排序的无解集合
}

// 初始化
func (p *Group) init() {
	p.All = []string{}
	p.lock = &sync.RWMutex{}
	p.lockclose = &sync.RWMutex{}
	p.m = map[string]*Fn{}
	p.closeset = []string{}
}

// Fork，参数hold指示需要跳过的执行器
//func (p *Group) Fork() *Group {
//	return p.fork()
//}
func (p *Group) fork(blacklist ...string) *Group {
	fork := &Group{}
	fork.init()
	sort.StringSlice(blacklist).Sort()
	p.lock.RLock()
	defer p.lock.RUnlock()
	for k, c := range p.m {
		if c.used || indexOf(blacklist, k) == -1 {
			d := (*c)
			fork.m[k] = &d
			fork.All = append(fork.All, k)
		}
	}
	if len(blacklist) == 0 {
		fork.closeset = make([]string, len(p.closeset))
		copy(fork.closeset, p.closeset)
	}
	sort.StringSlice(fork.All).Sort()
	return fork
}

// 把 k 存入无解集合
func (p *Group) pushCloseSet(eq string) {
	p.lockclose.Lock()
	defer p.lockclose.Unlock()
	if indexOf(p.closeset, eq) == -1 {
		p.closeset = append(p.closeset, eq)
		sort.StringSlice(p.closeset).Sort()
	}
}

//
func (p *Group) inCloseSet(eq string) bool {
	p.lockclose.RLock()
	defer p.lockclose.RUnlock()
	return indexOf(p.closeset, eq) != -1
}

// 注册执行函数, 如果不符合要求会直接抛出 panic.
func (p *Group) Register(fnlist ...interface{}) {
	if p.m == nil {
		p.init()
	}
	p.lockclose.RLock()
	defer p.lockclose.RUnlock()
	for _, fn := range fnlist {
		switch x := fn.(type) {
		default:
			p.register("", x)
		case FuncMap:
			for name, fn := range x {
				p.register(name, fn)
			}
		}
	}
	sort.StringSlice(p.All).Sort()
}

func (p *Group) register(name string, fn interface{}) {
	args, outs := proto.FuncSplit(fn)
	if len(outs) > 2 {
		outs = outs[1 : len(outs)-1]
	}
	args = args[1 : len(args)-1]
	fun := "func"
	if name != "" {
		fun += " " + name
	}
	fun += "(" + strings.Join(args, ", ") + ")"

	key := fun
	numout := len(outs)
	ify := ""
	if numout > 0 && (outs[numout-1] == "bool" || outs[numout-1] == "error") {
		numout--
		ify = outs[numout]
	}
	if numout == 1 {
		key += " " + outs[0]
	} else if numout > 1 {
		key += " (" + strings.Join(outs[:numout], ", ") + ")"
	}

	// 不允许注册相同 proto 的函数
	_, ok := p.m[key]
	if ok {
		panic("auto repeated: " + proto.Type(fn))
	}

	n := Fn{name: name}
	n.args = args
	n.outs = outs
	n.apply = reflect.ValueOf(fn)
	n.ify = ify
	n.numout = numout

	p.m[key] = &n
	p.All = append(p.All, key)
}

// 执行 val 到 like 类型, 并返回 interface{},args 是附加的参数
func (p *Group) To(like interface{}, args ...interface{}) (i interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.New(fmt.Sprint(e))
		}
	}()
	if len(args) == 0 {
		return nil, toInValidArgs("arguments length is zero")
	}

	c := p.match(like, args)

	if c == nil {
		return nil, toNotSupported(like)
	}
	// 单函数
	if len(c.queue) == 0 {
		in := []reflect.Value{}
		for _, arg := range args {
			in = append(in, proto.ValueOf(arg))
		}
		out := c.apply.Call(in)
		ify := len(out) - 1
		if ify < 0 || out[0].Kind().String() != proto.TypeOf(like).Kind().String() {
			return nil, toFail(like)
		}
		switch c.ify {
		case "bool":
			if !out[ify].Bool() {
				return nil, toFail(like)
			}
		case "error":
			if !out[ify].IsNil() {
				return nil, out[ify].Interface().(error)
			}
		}
		return out[0].Interface(), nil
	}

	// 队列
	var out []reflect.Value
	pos := 0
	for _, key := range c.queue {
		in := []reflect.Value{}
		fn := p.m[key]
		end := pos + len(fn.args) - len(out)

		for _, arg := range out {
			in = append(in, proto.ValueOf(arg))
		}

		for _, arg := range args[pos:end] {
			in = append(in, proto.ValueOf(arg))
		}

		pos = end

		out = fn.apply.Call(in)
		end = len(out) - 1
		if end < 0 {
			return nil, toFail(like)
		}
		switch fn.ify {
		case "bool":
			if out[end].Kind() != reflect.Bool || !out[end].Bool() {
				return nil, toFail(like)
			}
			out = out[:end]
		case "error":
			if !out[end].IsNil() {
				return nil, out[end].Interface().(error)
			}
			out = out[:end]
		}
	}
	return out[0].Interface(), nil
}

// 执行 val 到 to 类型, 并赋值到 to
func (p *Group) SetTo(to interface{}, args ...interface{}) (err error) {
	var val interface{}
	if len(args) > 0 {
		val = args[0]
	}
	v := proto.ValueOf(val)
	if !v.IsValid() {
		return toInValidArgs(v)
	}
	return nil
}

// 根据参数匹配,或者生成执行函数
func (p *Group) match(kind interface{}, arguments []interface{}) *Fn {
	to := proto.Type(kind)
	args := proto.Types(arguments...)
	// 快速匹配
	key := "func(" + strings.Join(args, ", ") + ") " + to
	fn, ok := p.m[key]
	if ok {
		return fn
	}
	// 无解
	if p.inCloseSet(key) {
		return nil
	}
	// 尝试生成新的序列
	np := p.fork()
	keys := np.npc(to, args)

	// 无解
	if len(keys) == 0 {
		p.pushCloseSet(key)
		return nil
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	// 在查找一次
	fn, ok = p.m[key]
	if ok {
		return fn
	}

	fn = &Fn{}
	fn.queue = keys
	p.m[key] = fn
	p.All = append(p.All, key)
	sort.StringSlice(p.All).Sort()
	return fn
}

func (p *Group) npc(to string, args []string) (key []string) {
	m := p.m
	a := p.All
	// 分离候选集合
	left := [][]int{}
	right := [][]int{}
	for idx, k := range a {
		c := m[k]
		if argsCompare(nil, args, c.args) && c.numout > 0 && c.outs[0] != to {
			left = append(left, []int{idx, 0})
		}
		right = append(right, []int{idx, 0})
	}
	min := len(right) / 2
	if min > 10 {
		min = 10
	}
	ok := false
	idx := -1
	for _, n := range left {
		fn := m[a[n[0]]]
		min, ok = p.npcWalk(to, fn.outs[:fn.numout], min, 0, right, args[len(fn.args):], n[0:1])
		if ok {
			idx = n[0]
		}
	}
	if idx == -1 {
		return
	}
	n := right[idx][2:]
	if len(n) == 0 {
		return
	}
	key = make([]string, len(n))
	for i, x := range n {
		key[i] = a[x]
	}
	return
}

// 比较输入类型匹配
func argsCompare(left, args, arg []string) bool {
	ll := len(left)
	la := len(args)
	l := len(arg)
	if ll+la < l {
		return false
	}
	i := 0
	for ; i < l && i < ll; i++ {
		if arg[i] != left[i] {
			return false
		}
	}

	for ; i < l && i < la; i++ {
		if arg[i] != args[i-ll] {
			return false
		}
	}
	return true
}

// 选择,从right中选出最小路径配对的函数
// 0 as O,1 as I
func (p *Group) npcWalk(totype string, leftype []string, min, deep int, right [][]int, args []string, order []int) (int, bool) {
	if deep >= min {
		return min, false
	}
	m := p.m
	a := p.All
	var ok, ret bool
	skip := order[len(order)-1]
	idx := order[0]
	deep++
	for i, n := range right {
		c := m[a[i]]
		if i == skip || i == idx || n[1] == -1 || (n[1] != 0 && n[1] < deep) {
			continue
		}
		// 过滤掉监视
		if c.numout == 0 {
			continue
		}

		// 参数匹配
		if !argsCompare(leftype, args, c.args) {
			continue
		}
		// 最终匹配
		if totype == c.outs[0] {
			// 都匹配了,参数还没有用完
			if len(args) != len(c.args)-1 {
				continue
			}
			n[1] = deep
			if deep < min { //只留下最小路径的
				//fmt.Println("NPC", deep, i, a[i], order)
				right[idx] = append(right[idx][:2], order...)
				right[idx] = append(right[idx], i)
				return deep, true
			} else {
				min = deep
				continue
			}
		}
		n[1] = deep
		min, ok = p.npcWalk(totype, c.outs[:c.numout], min, deep, right, args[len(c.args)-c.numout:], append(order, i))
		if ok {
			//return min, true
			ret = true
		}
	}
	return min, ret
}

//crossover,mutation
// 在已经排序的 []string 中查找 eq 的下标
func indexOf(a []string, eq string) int {
	i, j := 0, len(a)
	h := 0
	for i < j {
		h = i + (j-i)/2
		if a[h] == eq {
			return h
		}
		if h == i {
			break
		}
		if a[h] > eq {
			j = h
		} else {
			i = h
		}
	}
	return -1
}

// 初始化注册基本的类型执行
func init() {
	Conv.Register(
		func(i int8) (ii uint8, ok bool) {
			ii = uint8(i)
			if int8(ii) == i {
				ok = true
			}
			return
		},
		func(i uint8) (ii int8, ok bool) {
			ii = int8(i)
			if uint8(ii) == i {
				ok = true
			}
			return
		},
		func(i int64) (ii uint64, ok bool) {
			ii = uint64(i)
			if int64(ii) == i {
				ok = true
			}
			return
		},
		func(i uint64) (ii int64, ok bool) {
			ii = int64(i)
			if uint64(ii) == i {
				ok = true
			}
			return
		},
		func(i int8) int16 { return int16(i) },
		func(i int16) int32 { return int32(i) },
		func(i int32) int64 { return int64(i) },
		func(i int) int64 { return int64(i) },
		func(i int64) (ii int32, ok bool) {
			ii = int32(i)
			if int64(ii) == i {
				ok = true
			}
			return
		},
		func(i int64) (ii int, ok bool) {
			ii = int(i)
			if int64(ii) == i {
				ok = true
			}
			return
		},
		func(i int32) (ii int16, ok bool) {
			ii = int16(i)
			if int32(ii) == i {
				ok = true
			}
			return
		},
		func(i int16) (ii int8, ok bool) {
			ii = int8(i)
			if int16(ii) == i {
				ok = true
			}
			return
		},
		func(i uint8) uint16 { return uint16(i) },
		func(i uint16) uint32 { return uint32(i) },
		func(i uint32) uint64 { return uint64(i) },
		func(i uint64) (ii uint32, ok bool) {
			ii = uint32(i)
			if uint64(ii) == i {
				ok = true
			}
			return
		},
		func(i uint64) (ii uint, ok bool) {
			ii = uint(i)
			if uint64(ii) == i {
				ok = true
			}
			return
		},
		func(i uint32) (ii uint16, ok bool) {
			ii = uint16(i)
			if uint32(ii) == i {
				ok = true
			}
			return
		},
		func(i uint16) (ii uint8, ok bool) {
			ii = uint8(i)
			if uint16(ii) == i {
				ok = true
			}
			return
		},
		func(i int64) string {
			return strconv.FormatInt(i, 10)
		},
		func(i uint64) string {
			return strconv.FormatUint(i, 10)
		},
		strconv.Atoi,
		func(s string) (int64, error) {
			return strconv.ParseInt(s, 10, 64)
		},
		func(s string) (uint64, error) {
			return strconv.ParseUint(s, 10, 64)
		},
		func(s string) (float64, error) {
			return strconv.ParseFloat(s, 64)
		},
		strconv.ParseBool,
	)
}
