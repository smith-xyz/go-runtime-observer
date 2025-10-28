package instrumentlog

import (
	"os"
	"runtime"
	"sync"
)

const LOG_ENV_VAR = "INSTRUMENTATION_LOG_PATH"

var (
	logFile *os.File
	mu      sync.Mutex
	seen    = make(map[string]bool)
)

func init() {
	path := os.Getenv(LOG_ENV_VAR)
	if path != "" {
		logFile, _ = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	}
}

func LogCall(operation string, args ...string) {
	if logFile == nil {
		return
	}
	
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return
	}
	
	caller := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		caller = fn.Name()
	}
	
	key := operation + ":" + caller + ":" + file + ":" + itoa(line)
	for _, arg := range args {
		key += ":" + arg
	}
	
	mu.Lock()
	if seen[key] {
		mu.Unlock()
		return
	}
	seen[key] = true
	
	buf := make([]byte, 0, 256)
	buf = append(buf, "{\"operation\":\""...)
	buf = append(buf, operation...)
	buf = append(buf, "\",\"caller\":\""...)
	buf = append(buf, caller...)
	buf = append(buf, "\",\"file\":\""...)
	buf = append(buf, file...)
	buf = append(buf, "\",\"line\":"...)
	buf = append(buf, itoa(line)...)
	
	for i := 0; i+1 < len(args); i += 2 {
		buf = append(buf, ",\""...)
		buf = appendEscaped(buf, args[i])
		buf = append(buf, "\":\""...)
		buf = appendEscaped(buf, args[i+1])
		buf = append(buf, '"')
	}
	
	buf = append(buf, "}\n"...)
	
	_, _ = logFile.Write(buf)
	mu.Unlock()
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
