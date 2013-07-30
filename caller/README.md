## Caller

Chain arguments -- 论据链

最初实现 Caller 后, 没有找到一个合适的名字对这种方式进行命名. 最后找到了 Chain arguments 这个词, 这个词本来不是专为程序算法命名的. 原文在

[http://www.fallacyfiles.org/glossary.html](http://www.fallacyfiles.org/glossary.html)

`
Chain of arguments
    A series of arguments linked by the conclusion of each being a premiss in the next, except for the final argument in the chain. 
`

Caller 借用此词进行命名. 伟大的机器翻译成 `论据链`

### 举例

```go
package main

import (
	"fmt"
	"github.com/gohub/typeless/caller"
	"strconv"
)

func main() {
	fmt.Println(Ify("123456789"))
	fmt.Println(ChainArgs("123456789"))
}

// if 语句的方法
func Ify(input string) (uint64, bool) {
	if IsMinLen(input, 6) {
		out, err := strconv.ParseUint(input, 10, 64)
		if err == nil && IsRang(out, 1, 123456789) {
			return out, true
		}
	}
	return 0, false
}

// Chain arguments 的方法
func ChainArgs(input string) (v uint64, ok bool) {
	ok = caller.New().
		Push(
		IsMinLen, input, 6,
		strconv.ParseUint, input, 10, 64).
		OutTo(&v) == nil
	return
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
```

### 期待您的参与

目前 Caller 是实验性的, 还没有找到足够的理由和场景去应用.
或许 Caller 需要改进, 期待您的参与.