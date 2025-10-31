package instrumentlog

import (
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"
)

// ReceiverBaton represents the stable receiver pointer that survives transformations
// This is the "baton" passed between MethodByName and Call operations.
// See: https://github.com/smith-xyz/go-runtime-observer/blob/main/docs/correlation-algorithm.md#overview
type ReceiverBaton uintptr

var (
	tracker     *correlationTracker
	trackerOnce sync.Once
	debugFile   *os.File
	debugOnce   sync.Once
)

// CorrelationEntry stores method name and sequence for temporal ordering
// See: https://github.com/smith-xyz/go-runtime-observer/blob/main/docs/correlation-algorithm.md#algorithm-flow
type CorrelationEntry struct {
	MethodName  string
	ReceiverPtr uintptr
	SequenceNum uint64
	AccessTime  uint64
}

// correlationTracker implements the correlation map C[p] → [(name₁, seq₁), (name₂, seq₂), ...]
// See: https://github.com/smith-xyz/go-runtime-observer/blob/main/docs/correlation-algorithm.md#mathematical-notation
// Uses sync.Map for thread-safe access. See: https://github.com/smith-xyz/go-runtime-observer/blob/main/docs/correlation-algorithm.md#concurrency-safety
type correlationTracker struct {
	m               sync.Map // Maps ReceiverBaton (uintptr) → []*CorrelationEntry
	size            int64
	sequence        uint64
	maxEntries      int64
	maxAge          uint64
	cleanupInterval uint64
	evictions       int64
	matches         int64
	misses          int64
}

func initCorrelationTracker() {
	trackerOnce.Do(func() {
		maxEntries := getEnvIntCorrelation(ENV_MAX_CORRELATIONS, defaultMaxCorrelations)
		maxAge := getEnvUint64Correlation(ENV_CORRELATION_MAX_AGE, defaultCorrelationMaxAge)
		cleanupInterval := getEnvUint64Correlation(ENV_CLEANUP_INTERVAL, defaultCleanupInterval)

		tracker = &correlationTracker{
			maxEntries:      int64(maxEntries),
			maxAge:          maxAge,
			cleanupInterval: cleanupInterval,
		}

		go tracker.periodicCleanup()
	})
}

func getEnvIntCorrelation(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			return i
		}
	}
	return defaultValue
}

func getEnvUint64Correlation(key string, defaultValue uint64) uint64 {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.ParseUint(val, 10, 64); err == nil && i > 0 {
			return i
		}
	}
	return defaultValue
}

func initDebugFile() {
	debugOnce.Do(func() {
		if os.Getenv(ENV_DEBUG_CORRELATION) == "true" {
			path := os.Getenv("INSTRUMENTATION_DEBUG_LOG_PATH")
			if path == "" {
				path = "/tmp/instrumentation-correlation-debug.log"
			}
			var err error
			debugFile, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				debugFile = nil
			}
		}
	})
}

func debugWrite(msg string) {
	if debugFile == nil {
		return
	}
	buf := []byte(msg)
	buf = append(buf, '\n')
	_, _ = debugFile.Write(buf)
}

func debugFormatUint64(i uint64) string {
	return formatUint64Correlation(i)
}

func formatUint64Correlation(i uint64) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	n := len(buf)
	for i > 0 {
		n--
		buf[n] = byte(i%10) + '0'
		i /= 10
	}
	return string(buf[n:])
}

// RecordMethodByName records a correlation when MethodByName or Method is called
// Implements Phase 1: Recording from https://github.com/smith-xyz/go-runtime-observer/blob/main/docs/correlation-algorithm.md#phase-1-recording-methodbyname-time
// Process:
//  1. Extract receiver pointer (baton): p = ptr(receiverValue)  // See "Key Invariant"
//  2. Create correlation entry: (name, seq++)
//  3. Store: C[p] = [entry] + C[p] (prepend, keep last 10)  // See "Performance" for bounded growth
func RecordMethodByName(methodValue any, methodName string, receiverValue any) {
	initCorrelationTracker()
	initDebugFile()

	methodValuePtr := extractValuePtr(methodValue)
	if methodValuePtr == 0 {
		debugWrite("RECORD: methodValuePtr=0 (extraction failed)")
		return
	}

	baton := ReceiverBaton(extractValuePtr(receiverValue))
	if baton == 0 {
		debugWrite("RECORD: baton=0 (extraction failed)")
		return
	}

	seq := atomic.AddUint64(&tracker.sequence, 1)
	currentSize := atomic.LoadInt64(&tracker.size)

	if currentSize >= tracker.maxEntries {
		evictCount := int(tracker.maxEntries / 10)
		if evictCount == 0 {
			evictCount = 1
		}
		tracker.evictLRU(evictCount)
	}

	entry := &CorrelationEntry{
		MethodName:  methodName,
		ReceiverPtr: uintptr(baton),
		SequenceNum: seq,
		AccessTime:  seq,
	}

	val, loaded := tracker.m.LoadOrStore(uintptr(baton), []*CorrelationEntry{})
	entries, ok := val.([]*CorrelationEntry)
	if !ok {
		entries = []*CorrelationEntry{}
	}

	newEntries := make([]*CorrelationEntry, 0, len(entries)+1)
	newEntries = append(newEntries, entry)
	newEntries = append(newEntries, entries...)

	if len(newEntries) > 10 {
		newEntries = newEntries[:10]
	}

	tracker.m.Store(uintptr(baton), newEntries)

	if !loaded {
		atomic.AddInt64(&tracker.size, 1)
	}

	debugWrite("RECORD: methodValuePtr=" + debugFormatUint64(uint64(methodValuePtr)) +
		" methodName=" + methodName +
		" baton=" + debugFormatUint64(uint64(baton)) +
		" seq=" + debugFormatUint64(seq) +
		" entryCount=" + formatUint64Correlation(uint64(len(newEntries))))

	if seq%tracker.cleanupInterval == 0 {
		go tracker.cleanupByAge()
	}
}

// GetCorrelation looks up a correlation when Call or CallSlice is invoked
// Implements Phase 2: Lookup from https://github.com/smith-xyz/go-runtime-observer/blob/main/docs/correlation-algorithm.md#phase-2-lookup-call-time
// Process:
//  1. Extract receiver pointer (baton): p = ptr(callReceiverValue)
//  2. Lookup: C[p]
//  3. If found: return most recent entry, consume from map
//  4. If not found: return nil (fallback handled separately if needed)
//
// See: https://github.com/smith-xyz/go-runtime-observer/blob/main/docs/correlation-algorithm.md#transformation-survival-matrix
func GetCorrelation(callReceiverValue any) (*CorrelationEntry, bool) {
	initCorrelationTracker()
	initDebugFile()

	baton := ReceiverBaton(extractValuePtr(callReceiverValue))
	if baton == 0 {
		debugWrite("GET: baton=0 (extraction failed)")
		atomic.AddInt64(&tracker.misses, 1)
		return nil, false
	}

	val, ok := tracker.m.Load(uintptr(baton))
	if ok {
		entries, ok := val.([]*CorrelationEntry)
		if !ok || len(entries) == 0 {
			atomic.AddInt64(&tracker.misses, 1)
			debugWrite("GET: baton=" + debugFormatUint64(uint64(baton)) + " MISS (invalid type or empty)")
			return nil, false
		}

		entry := entries[0]

		if len(entries) > 1 {
			tracker.m.Store(uintptr(baton), entries[1:])
		} else {
			tracker.m.Delete(uintptr(baton))
			atomic.AddInt64(&tracker.size, -1)
		}

		atomic.AddInt64(&tracker.matches, 1)
		debugWrite("GET: baton=" + debugFormatUint64(uint64(baton)) +
			" MATCH methodName=" + entry.MethodName +
			" seq=" + debugFormatUint64(entry.SequenceNum))
		return entry, true
	}

	atomic.AddInt64(&tracker.misses, 1)
	debugWrite("GET: baton=" + debugFormatUint64(uint64(baton)) + " MISS")
	return nil, false
}

// GetCorrelationFromPtr is a convenience function for lookup using a uintptr directly
// Used when the receiver pointer is already extracted as a string and parsed
func GetCorrelationFromPtr(callReceiverPtr uintptr) (*CorrelationEntry, bool) {
	initCorrelationTracker()
	initDebugFile()

	baton := ReceiverBaton(callReceiverPtr)
	if baton == 0 {
		debugWrite("GET: baton=0 (extraction failed)")
		atomic.AddInt64(&tracker.misses, 1)
		return nil, false
	}

	val, ok := tracker.m.Load(uintptr(baton))
	if ok {
		entries, ok := val.([]*CorrelationEntry)
		if !ok || len(entries) == 0 {
			atomic.AddInt64(&tracker.misses, 1)
			debugWrite("GET: baton=" + debugFormatUint64(uint64(baton)) + " MISS (invalid type or empty)")
			return nil, false
		}

		entry := entries[0]

		if len(entries) > 1 {
			tracker.m.Store(uintptr(baton), entries[1:])
		} else {
			tracker.m.Delete(uintptr(baton))
			atomic.AddInt64(&tracker.size, -1)
		}

		atomic.AddInt64(&tracker.matches, 1)
		debugWrite("GET: baton=" + debugFormatUint64(uint64(baton)) +
			" MATCH methodName=" + entry.MethodName +
			" seq=" + debugFormatUint64(entry.SequenceNum))
		return entry, true
	}

	atomic.AddInt64(&tracker.misses, 1)
	debugWrite("GET: baton=" + debugFormatUint64(uint64(baton)) + " MISS")
	return nil, false
}

// extractValuePtr extracts the internal ptr field from reflect.Value
// This implements the ptr() function from https://github.com/smith-xyz/go-runtime-observer/blob/main/docs/correlation-algorithm.md#mathematical-notation
// The receiver pointer (baton) survives transformations per the "Key Invariant"
func extractValuePtr(v any) uintptr {
	if v == nil {
		return 0
	}

	type iface struct {
		typ unsafe.Pointer
		ptr unsafe.Pointer
	}

	valIface := (*iface)(unsafe.Pointer(&v))
	if valIface.ptr == nil {
		return 0
	}

	type valueHeader struct {
		typ unsafe.Pointer
		ptr unsafe.Pointer
	}

	valueStruct := (*valueHeader)(valIface.ptr)
	if valueStruct.ptr == nil {
		return 0
	}

	return uintptr(valueStruct.ptr)
}

// evictLRU removes the least recently used entries when the map reaches capacity
// See: https://github.com/smith-xyz/go-runtime-observer/blob/main/docs/correlation-algorithm.md#performance
func (ct *correlationTracker) evictLRU(count int) {
	type entryWithTime struct {
		ptr  uintptr
		time uint64
	}

	entries := make([]entryWithTime, 0, count*2)

	ct.m.Range(func(key, value interface{}) bool {
		entrySlice, ok := value.([]*CorrelationEntry)
		if !ok || len(entrySlice) == 0 {
			return true
		}
		ptr, ok := key.(uintptr)
		if !ok {
			return true
		}
		entry := entrySlice[0]
		entries = append(entries, entryWithTime{
			ptr:  ptr,
			time: entry.AccessTime,
		})
		return len(entries) < count*2
	})

	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].time < entries[j-1].time; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}

	evicted := 0
	for _, e := range entries {
		if evicted >= count {
			break
		}
		if _, ok := ct.m.LoadAndDelete(e.ptr); ok {
			evicted++
		}
	}

	atomic.AddInt64(&ct.size, -int64(evicted))
	atomic.AddInt64(&ct.evictions, int64(evicted))
}

// cleanupByAge removes entries older than maxAge based on sequence number
// See: https://github.com/smith-xyz/go-runtime-observer/blob/main/docs/correlation-algorithm.md#performance
func (ct *correlationTracker) cleanupByAge() {
	currentSeq := atomic.LoadUint64(&ct.sequence)
	if currentSeq < ct.maxAge {
		return
	}
	cutoff := currentSeq - ct.maxAge

	deleted := 0
	keysToDelete := make([]uintptr, 0, 100)
	ct.m.Range(func(key, value interface{}) bool {
		entrySlice, ok := value.([]*CorrelationEntry)
		if !ok || len(entrySlice) == 0 {
			return true
		}
		ptr, ok := key.(uintptr)
		if !ok {
			return true
		}
		firstEntry := entrySlice[0]
		if firstEntry.SequenceNum < cutoff {
			keysToDelete = append(keysToDelete, ptr)
		}
		return len(keysToDelete) < 1000
	})

	for _, key := range keysToDelete {
		if _, ok := ct.m.LoadAndDelete(key); ok {
			deleted++
		}
	}

	if deleted > 0 {
		atomic.AddInt64(&ct.size, -int64(deleted))
	}
}

func (ct *correlationTracker) periodicCleanup() {
	for {
		runtime.Gosched()

		currentSeq := atomic.LoadUint64(&ct.sequence)
		targetSeq := currentSeq + ct.cleanupInterval

		for atomic.LoadUint64(&ct.sequence) < targetSeq {
			runtime.Gosched()
		}

		ct.cleanupByAge()
	}
}

func GetCorrelationMetrics() map[string]int64 {
	initCorrelationTracker()

	return map[string]int64{
		"size":      atomic.LoadInt64(&tracker.size),
		"matches":   atomic.LoadInt64(&tracker.matches),
		"misses":    atomic.LoadInt64(&tracker.misses),
		"evictions": atomic.LoadInt64(&tracker.evictions),
	}
}
