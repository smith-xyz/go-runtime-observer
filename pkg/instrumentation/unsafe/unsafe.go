package unsafe

import (
	"fmt"
	"unsafe"

	"github.com/smith-xyz/go-runtime-observer/pkg/instrumentation/instrumentlog"
)

type Pointer = unsafe.Pointer
type IntegerType = int

func Add(ptr Pointer, len IntegerType) Pointer {
	instrumentlog.LogCall("unsafe.Add", "ptr", fmt.Sprintf("%p", ptr), "len", fmt.Sprintf("%d", len))
	return unsafe.Add(ptr, len)
}
