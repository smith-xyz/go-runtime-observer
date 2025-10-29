package unsafe

import (
	"fmt"
	"unsafe"

	"github.com/smith-xyz/go-runtime-observer/pkg/instrumentation/instrumentlog"
)

type IntegerType = int

func Add(ptr unsafe.Pointer, len IntegerType) unsafe.Pointer {
	instrumentlog.LogCall("unsafe.Add", "ptr", fmt.Sprintf("%p", ptr), "len", fmt.Sprintf("%d", len))
	return unsafe.Add(ptr, len)
}

func Slice(ptr *byte, len IntegerType) []byte {
	instrumentlog.LogCall("unsafe.Slice", "ptr", fmt.Sprintf("%p", ptr), "len", fmt.Sprintf("%d", len))
	return unsafe.Slice(ptr, len)
}
