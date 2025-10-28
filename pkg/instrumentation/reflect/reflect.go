package reflect

import (
	"fmt"
	"reflect"

	"github.com/smith-xyz/go-runtime-observer/pkg/instrumentation/instrumentlog"
)

func ValueOf(i any) reflect.Value {
	instrumentlog.LogCall("reflect.ValueOf", "type", fmt.Sprintf("%T", i))
	return reflect.ValueOf(i)
}
