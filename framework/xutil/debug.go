package xutil

import (
	"fmt"
	"runtime"
	"strings"
)

// 打印堆栈信息
func StackTrace(err interface{}, indent string) string {
	var stack strings.Builder

	fmt.Fprint(&stack, "\n")
	fmt.Fprintf(&stack, "%sError Message: %v\n", indent, err)

	for i := 1; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fmt.Fprintf(&stack, "%s%s:%d (0x%x)\n", indent, file, line, pc)
	}

	return stack.String()
}
