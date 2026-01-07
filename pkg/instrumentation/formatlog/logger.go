package formatlog

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	LogEnvVar              = "INSTRUMENTATION_LOG_PATH"
	EnvMaxSeenEntries      = "INSTRUMENTATION_MAX_SEEN_ENTRIES"
	defaultMaxSeenEntries  = 500000
	callerSkipDepth        = 3
	instrumentationPattern = "runtime_observe_instrumentation"
)

var (
	logFile  *os.File
	mu       sync.Mutex
	seen     = make(map[string]bool)
	seenSize int64
	initOnce sync.Once
)

func initLogger() {
	initOnce.Do(func() {
		path := os.Getenv(LogEnvVar)
		if path != "" {
			logFile, _ = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		}
	})
}

type CallArgs map[string]string

type LogCallBuilder struct {
	operation string
	args      CallArgs
}

func NewLogCall(operation string) *LogCallBuilder {
	initLogger()
	return &LogCallBuilder{
		operation: operation,
		args:      make(CallArgs),
	}
}

func (b *LogCallBuilder) Add(name string, value string) *LogCallBuilder {
	b.args[name] = value
	return b
}

func (b *LogCallBuilder) Log() {
	if logFile == nil {
		return
	}

	pc, file, line, ok := runtime.Caller(callerSkipDepth)
	if !ok {
		return
	}

	if strings.Contains(file, instrumentationPattern) {
		return
	}

	caller := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		caller = fn.Name()
		if strings.Contains(caller, instrumentationPattern) {
			return
		}
	}

	key := b.operation + ":" + caller + ":" + file + ":" + fmt.Sprintf("%d", line)
	for name, value := range b.args {
		key += ":" + name + ":" + value
	}

	mu.Lock()
	currentSeenSize := atomic.LoadInt64(&seenSize)
	maxEntries := getEnvInt(EnvMaxSeenEntries, defaultMaxSeenEntries)
	if currentSeenSize < int64(maxEntries) {
		if seen[key] {
			mu.Unlock()
			return
		}
		seen[key] = true
		atomic.AddInt64(&seenSize, 1)
	}
	mu.Unlock()

	buf := make([]byte, 0, 512)
	buf = append(buf, `{"operation":"`...)
	buf = append(buf, b.operation...)
	buf = append(buf, '"')

	for name, value := range b.args {
		buf = append(buf, `,"`...)
		buf = appendEscaped(buf, name)
		buf = append(buf, `":"`...)
		buf = appendEscaped(buf, value)
		buf = append(buf, '"')
	}

	buf = append(buf, `,"caller":"`...)
	buf = append(buf, caller...)
	buf = append(buf, `","file":"`...)
	buf = append(buf, file...)
	buf = append(buf, `","line":`...)
	buf = append(buf, fmt.Sprintf("%d", line)...)
	buf = append(buf, "}\n"...)

	mu.Lock()
	_, _ = logFile.Write(buf)
	mu.Unlock()
}

func LogCall(operation string, args CallArgs) {
	builder := NewLogCall(operation)
	for name, value := range args {
		builder.Add(name, value)
	}
	builder.Log()
}

func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		var i int
		if _, err := fmt.Sscanf(val, "%d", &i); err == nil && i > 0 {
			return i
		}
	}
	return defaultValue
}

func FormatInt(i int) string         { return fmt.Sprintf("%d", i) }
func FormatInt32(i int32) string     { return fmt.Sprintf("%d", i) }
func FormatInt64(i int64) string     { return fmt.Sprintf("%d", i) }
func FormatUint(u uint) string       { return fmt.Sprintf("%d", u) }
func FormatUint32(u uint32) string   { return fmt.Sprintf("%d", u) }
func FormatUint64(u uint64) string   { return fmt.Sprintf("%d", u) }
func FormatFloat64(f float64) string { return fmt.Sprintf("%g", f) }
func FormatBool(b bool) string       { return fmt.Sprintf("%t", b) }
func FormatString(s string) string   { return s }
func FormatPointer(v any) string     { return fmt.Sprintf("%p", v) }

func FormatBytes(b []byte) string {
	if len(b) == 0 {
		return "len:0"
	}
	if len(b) <= 8 {
		return fmt.Sprintf("len:%d,hex:%x", len(b), b)
	}
	return fmt.Sprintf("len:%d,hex:%x..", len(b), b[:8])
}

func FormatAny(v any) string {
	if v == nil {
		return "nil"
	}
	switch x := v.(type) {
	case string:
		if len(x) > 100 {
			return x[:100] + "..."
		}
		return x
	case []byte:
		return FormatBytes(x)
	case error:
		return x.Error()
	case fmt.Stringer:
		return x.String()
	default:
		return formatReflect(v)
	}
}

func formatReflect(v any) string {
	rv := reflect.ValueOf(v)
	rt := reflect.TypeOf(v)

	if rt.Kind() == reflect.Interface || rt.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return "nil"
		}
		return formatInterfaceValue(rv, rt)
	}

	return fmt.Sprintf("%+v", v)
}

func formatInterfaceValue(rv reflect.Value, rt reflect.Type) string {
	concreteType := rt.String()

	if method := rv.MethodByName("Name"); method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() >= 1 {
		if result := safeCallMethod(method); result != "" {
			return fmt.Sprintf("%s(%s)", concreteType, result)
		}
	}

	if method := rv.MethodByName("Params"); method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() >= 1 {
		results := method.Call(nil)
		if len(results) > 0 && results[0].Kind() == reflect.Ptr && !results[0].IsNil() {
			paramsVal := results[0].Elem()
			if nameField := paramsVal.FieldByName("Name"); nameField.IsValid() && nameField.Kind() == reflect.String {
				return fmt.Sprintf("%s(%s)", concreteType, nameField.String())
			}
		}
	}

	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		elem := rv.Elem()
		if elem.Kind() == reflect.Struct {
			return fmt.Sprintf("%s{%+v}", concreteType, elem.Interface())
		}
	}

	return fmt.Sprintf("%s@%p", concreteType, rv.Interface())
}

func safeCallMethod(method reflect.Value) string {
	defer func() { recover() }()
	results := method.Call(nil)
	if len(results) > 0 {
		if s, ok := results[0].Interface().(string); ok {
			return s
		}
		return fmt.Sprintf("%v", results[0].Interface())
	}
	return ""
}

func FormatValue(v any) string {
	return FormatAny(v)
}

func appendEscaped(buf []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			buf = append(buf, '\\', '"')
		case '\\':
			buf = append(buf, '\\', '\\')
		case '\n':
			buf = append(buf, '\\', 'n')
		case '\r':
			buf = append(buf, '\\', 'r')
		case '\t':
			buf = append(buf, '\\', 't')
		default:
			if c < 0x20 {
				buf = append(buf, '\\', 'u', '0', '0', hexDigit(c>>4), hexDigit(c&0xf))
			} else {
				buf = append(buf, c)
			}
		}
	}
	return buf
}

func hexDigit(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'a' + b - 10
}
