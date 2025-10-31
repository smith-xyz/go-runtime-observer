package unsafe

import (
	"fmt"
	"unsafe"

	"github.com/smith-xyz/go-runtime-observer/pkg/instrumentation/instrumentlog"
)

type IntegerType = int

func Add(ptr unsafe.Pointer, len IntegerType) unsafe.Pointer {
	instrumentlog.LogCall("unsafe.Add", instrumentlog.CallArgs{
		"ptr": fmt.Sprintf("%p", ptr),
		"len": fmt.Sprintf("%d", len),
	})
	return unsafe.Add(ptr, len)
}

func Slice(ptr *byte, len IntegerType) []byte {
	instrumentlog.LogCall("unsafe.Slice", instrumentlog.CallArgs{
		"ptr": fmt.Sprintf("%p", ptr),
		"len": fmt.Sprintf("%d", len),
	})
	return unsafe.Slice(ptr, len)
}

func SliceData(slice []byte) *byte {
	instrumentlog.LogCall("unsafe.SliceData", instrumentlog.CallArgs{
		"len": fmt.Sprintf("%d", len(slice)),
	})
	return unsafe.SliceData(slice)
}

func String(ptr *byte, len IntegerType) string {
	instrumentlog.LogCall("unsafe.String", instrumentlog.CallArgs{
		"ptr": fmt.Sprintf("%p", ptr),
		"len": fmt.Sprintf("%d", len),
	})
	return unsafe.String(ptr, len)
}

func StringData(str string) *byte {
	instrumentlog.LogCall("unsafe.StringData", instrumentlog.CallArgs{
		"len": fmt.Sprintf("%d", len(str)),
	})
	return unsafe.StringData(str)
}
