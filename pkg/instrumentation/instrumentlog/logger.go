package instrumentlog

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	logFile  *os.File
	mu       sync.Mutex
	seen     = make(map[string]bool)
	seenSize int64
)

func init() {
	path := os.Getenv(LOG_ENV_VAR)
	if path != "" {
		// Use 0600 (owner-only) for better security
		// Logs contain memory addresses and file paths - restrict access
		logFile, _ = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	}
	initCorrelationTracker()
}

func getEnvIntSeen(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			return i
		}
	}
	return defaultValue
}

type CallArgs map[string]string

type LogCallBuilder struct {
	operation string
	args      CallArgs
}

func NewLogCall(operation string) *LogCallBuilder {
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

	caller := defaultCallerName
	if fn := runtime.FuncForPC(pc); fn != nil {
		caller = fn.Name()
		if strings.Contains(caller, instrumentationPattern) {
			return
		}
	}

	key := b.operation + ":" + caller + ":" + file + ":" + itoa(line)
	for name, value := range b.args {
		key += ":" + name + ":" + value
	}

	mu.Lock()
	currentSeenSize := atomic.LoadInt64(&seenSize)
	if currentSeenSize < int64(getEnvIntSeen(ENV_MAX_SEEN_ENTRIES, defaultMaxSeenEntries)) {
		if seen[key] {
			mu.Unlock()
			return
		}
		seen[key] = true
		atomic.AddInt64(&seenSize, 1)
	}
	mu.Unlock()

	buf := make([]byte, 0, logBufferInitialSize)
	buf = append(buf, "{\"operation\":\""...)
	buf = append(buf, b.operation...)
	buf = append(buf, '"')

	// Add arguments first (name/value pairs)
	for name, value := range b.args {
		buf = append(buf, ",\""...)
		buf = appendEscaped(buf, name)
		buf = append(buf, "\":\""...)
		buf = appendEscaped(buf, value)
		buf = append(buf, '"')
	}

	// Then add caller, file, line
	buf = append(buf, ",\"caller\":\""...)
	buf = append(buf, caller...)
	buf = append(buf, "\",\"file\":\""...)
	buf = append(buf, file...)
	buf = append(buf, "\",\"line\":"...)
	buf = append(buf, itoa(line)...)

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

	receiverType := extractReceiverFromOperation(operation)
	debugLogCorrelationCheck(operation, receiverType, len(args))

	if shouldPerformCorrelationLookup(args) && len(args) > 0 {
		receiverValue, ok := args["v"]
		if ok && receiverValue != "" {
			var receiverPtr uintptr
			if ptr, err := strconv.ParseUint(receiverValue, 10, 64); err == nil {
				receiverPtr = uintptr(ptr)
			}

			if receiverPtr != 0 {
				debugLogCorrelationLookup(operation, receiverPtr, receiverValue)
				correlation, found := GetCorrelationFromPtr(receiverPtr)
				if found {
					builder.Add("method_name", correlation.MethodName)
					builder.Add("correlation_seq", FormatUint64(correlation.SequenceNum))
				}
			} else {
				debugLogCorrelationCheck(operation, "receiverPtr=0 after parse", 0)
			}
		}
	}

	builder.Log()
}

func debugLogCorrelationCheck(operation string, receiverType string, argCount int) {
	if os.Getenv("INSTRUMENTATION_DEBUG_CORRELATION") != "true" {
		return
	}
	path := os.Getenv("INSTRUMENTATION_DEBUG_LOG_PATH")
	if path == "" {
		path = "/tmp/instrumentation-correlation-debug.log"
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer file.Close()

	msg := "CHECK: operation=" + operation +
		" receiverType=" + receiverType +
		" argCount=" + FormatInt(argCount)
	buf := []byte(msg)
	buf = append(buf, '\n')
	_, _ = file.Write(buf)
}

func debugLogCorrelationLookup(operation string, receiverPtr uintptr, receiverValue string) {
	if os.Getenv("INSTRUMENTATION_DEBUG_CORRELATION") != "true" {
		return
	}
	path := os.Getenv("INSTRUMENTATION_DEBUG_LOG_PATH")
	if path == "" {
		path = "/tmp/instrumentation-correlation-debug.log"
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer file.Close()

	msg := "LOOKUP: operation=" + operation +
		" receiverValue=" + receiverValue +
		" receiverPtr=" + FormatUint64(uint64(receiverPtr))
	buf := []byte(msg)
	buf = append(buf, '\n')
	_, _ = file.Write(buf)
}

func shouldPerformCorrelationLookup(args CallArgs) bool {
	_, ok := args["_correlation_lookup"]
	return ok
}

func extractReceiverFromOperation(operation string) string {
	parts := strings.Split(operation, ".")
	if len(parts) < 3 {
		return ""
	}
	return parts[len(parts)-2]
}

func appendEscaped(dst []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' || c == '\\' {
			dst = append(dst, '\\', c)
		} else if c == '\n' {
			dst = append(dst, '\\', 'n')
		} else if c < 0x20 {
			dst = append(dst, '\\', 'u', '0', '0',
				"0123456789abcdef"[c>>4],
				"0123456789abcdef"[c&0xf])
		} else {
			dst = append(dst, c)
		}
	}
	return dst
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var buf [20]byte
	n := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}

	for i > 0 {
		n--
		buf[n] = byte(i%10) + '0'
		i /= 10
	}

	if neg {
		n--
		buf[n] = '-'
	}

	return string(buf[n:])
}

// FormatValue extracts the internal ptr field from reflect.Value for unique identification
// This allows tracking the same Value instance across struct copies (e.g., MethodByName -> Call)
// The v parameter must be a reflect.Value (we're in the reflect package, so this is safe)
func FormatValue(v any) string {
	if v == nil {
		return "nil"
	}

	// When v is passed as any, it's wrapped in an interface{}
	// We need to extract the actual Value struct from the interface
	// Interface layout: { type *runtime._type, data unsafe.Pointer }
	type iface struct {
		typ unsafe.Pointer
		ptr unsafe.Pointer
	}

	valIface := (*iface)(unsafe.Pointer(&v))
	if valIface.ptr == nil {
		return "nil"
	}

	// Now valIface.ptr points to the actual Value struct
	// Value struct layout: { typ_ *abi.Type, ptr unsafe.Pointer, flag uintptr }
	// We need to offset past typ_ (one pointer size) to get ptr field
	type valueHeader struct {
		typ unsafe.Pointer
		ptr unsafe.Pointer
	}

	valueStruct := (*valueHeader)(valIface.ptr)
	if valueStruct.ptr == nil {
		return "nil"
	}

	addr := uintptr(valueStruct.ptr)
	return FormatUint64(uint64(addr))
}

func FormatPointer(v any) string {
	if v == nil {
		return "nil"
	}
	ptr := unsafe.Pointer(&v)
	addr := uintptr(ptr)
	return FormatUint64(uint64(addr))
}

func FormatInt(i int) string {
	return itoa(i)
}

func FormatInt32(i int32) string {
	return itoa(int(i))
}

func FormatInt64(i int64) string {
	return itoa(int(i))
}

func FormatUint64(i uint64) string {
	return formatUint(i)
}

func FormatUint32(i uint32) string {
	return formatUint(uint64(i))
}

func FormatUint(i uint) string {
	return formatUint(uint64(i))
}

func FormatFloat64(f float64) string {
	return formatFloat(f)
}

func FormatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func FormatString(s string) string {
	return s
}

func FormatBytes(b []byte) string {
	if len(b) == 0 {
		return "len:0"
	}
	previewLen := 8
	if len(b) < previewLen {
		previewLen = len(b)
	}
	lenStr := formatUint(uint64(len(b)))
	totalLen := 4 + len(lenStr) + 5 + previewLen*2 + 2
	preview := make([]byte, 0, totalLen)
	preview = append(preview, 'l', 'e', 'n', ':')
	for i := 0; i < len(lenStr); i++ {
		preview = append(preview, lenStr[i])
	}
	preview = append(preview, ',', 'h', 'e', 'x', ':')
	for i := 0; i < previewLen; i++ {
		hex := "0123456789abcdef"
		preview = append(preview, hex[b[i]>>4], hex[b[i]&0xf])
	}
	if len(b) > previewLen {
		preview = append(preview, '.', '.')
	}
	return string(preview)
}

func FormatAny(v any) string {
	if v == nil {
		return "nil"
	}
	// Type assertions without reflect to avoid import cycle
	switch x := v.(type) {
	case string:
		return FormatString(x)
	case int:
		return FormatInt(x)
	case int8:
		return FormatInt(int(x))
	case int16:
		return FormatInt(int(x))
	case int32:
		return FormatInt(int(x))
	case int64:
		return FormatInt64(x)
	case uint:
		return FormatUint(x)
	case uint8:
		return FormatUint(uint(x))
	case uint16:
		return FormatUint(uint(x))
	case uint32:
		return FormatUint(uint(x))
	case uint64:
		return FormatUint64(x)
	case uintptr:
		return FormatUint64(uint64(x))
	case float32:
		return FormatFloat64(float64(x))
	case float64:
		return FormatFloat64(x)
	case bool:
		return FormatBool(x)
	case []byte:
		return FormatBytes(x)
	case []string:
		return "slice:string,len:" + FormatInt(len(x))
	case []int:
		return "slice:int,len:" + FormatInt(len(x))
	case []int64:
		return "slice:int64,len:" + FormatInt(len(x))
	case []uint:
		return "slice:uint,len:" + FormatInt(len(x))
	case []uint64:
		return "slice:uint64,len:" + FormatInt(len(x))
	case []float64:
		return "slice:float64,len:" + FormatInt(len(x))
	case []any:
		return "slice:any,len:" + FormatInt(len(x))
	default:
		// Complex types: use pointer address for unique instance tracking
		return FormatPointer(v)
	}
}

func formatUint(u uint64) string {
	if u == 0 {
		return "0"
	}
	var buf [20]byte
	n := len(buf)
	for u > 0 {
		n--
		buf[n] = byte(u%10) + '0'
		u /= 10
	}
	return string(buf[n:])
}

func formatFloat(f float64) string {
	if f == 0 {
		return "0"
	}
	neg := f < 0
	if neg {
		f = -f
	}

	var buf [64]byte
	n := len(buf)

	whole := uint64(f)
	frac := f - float64(whole)

	if frac == 0 {
		result := formatUint(whole)
		if neg {
			return "-" + result
		}
		return result
	}

	for i := 0; i < 6 && frac > 0; i++ {
		frac *= 10
		digit := int(frac)
		n--
		buf[n] = byte(digit) + '0'
		frac -= float64(digit)
	}

	n--
	buf[n] = '.'

	if whole == 0 {
		n--
		buf[n] = '0'
	} else {
		for whole > 0 {
			n--
			buf[n] = byte(whole%10) + '0'
			whole /= 10
		}
	}

	if neg {
		n--
		buf[n] = '-'
	}

	return string(buf[n:])
}
