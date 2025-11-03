package unsafe

import (
	"fmt"
	"unsafe"

	"github.com/smith-xyz/go-runtime-observer/pkg/instrumentation/instrumentlog"
)

type IntegerType interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func convertToInt[T IntegerType](v T) int {
	switch any(v).(type) {
	case int:
		return int(v)
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case uintptr:
		return int(v)
	default:
		return 0
	}
}

func Add[T IntegerType](ptr unsafe.Pointer, len T) unsafe.Pointer {
	instrumentlog.LogCall("unsafe.Add", instrumentlog.CallArgs{
		"ptr": fmt.Sprintf("%p", ptr),
		"len": fmt.Sprintf("%v", len),
	})
	return unsafe.Add(ptr, convertToInt(len))
}

func Slice[T any, L IntegerType](ptr *T, len L) []T {
	instrumentlog.LogCall("unsafe.Slice", instrumentlog.CallArgs{
		"ptr": fmt.Sprintf("%p", ptr),
		"len": fmt.Sprintf("%v", len),
	})
	return unsafe.Slice(ptr, convertToInt(len))
}

func SliceData[T any](slice []T) *T {
	instrumentlog.LogCall("unsafe.SliceData", instrumentlog.CallArgs{
		"len": fmt.Sprintf("%d", len(slice)),
	})
	return unsafe.SliceData(slice)
}

func String[T IntegerType](ptr *byte, len T) string {
	instrumentlog.LogCall("unsafe.String", instrumentlog.CallArgs{
		"ptr": fmt.Sprintf("%p", ptr),
		"len": fmt.Sprintf("%v", len),
	})
	return unsafe.String(ptr, convertToInt(len))
}

func StringData(str string) *byte {
	instrumentlog.LogCall("unsafe.StringData", instrumentlog.CallArgs{
		"len": fmt.Sprintf("%d", len(str)),
	})
	return unsafe.StringData(str)
}
