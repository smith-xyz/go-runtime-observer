package correlation

import (
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRecordMethodByName_WithBaton(t *testing.T) {
	resetTracker()

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)
	methodValue := reflect.ValueOf(obj)

	RecordMethodByName(methodValue, "GetName", receiverValue)

	baton := ReceiverBaton(ExtractValuePtr(receiverValue))
	entry, found := GetCorrelation(receiverValue)

	if !found {
		t.Fatal("Expected correlation to be found")
	}

	if entry.MethodName != "GetName" {
		t.Errorf("Expected method name 'GetName', got %s", entry.MethodName)
	}

	if ReceiverBaton(entry.ReceiverPtr) != baton {
		t.Errorf("Expected receiverPtr to match baton: got %d, want %d", entry.ReceiverPtr, uintptr(baton))
	}

	if entry.SequenceNum == 0 {
		t.Error("Expected sequence number to be non-zero")
	}
}

func TestRecordMethodByName_MultipleEntriesPerReceiver(t *testing.T) {
	resetTracker()

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)
	methodValue1 := reflect.ValueOf(obj)
	methodValue2 := reflect.ValueOf(obj)

	RecordMethodByName(methodValue1, "GetName", receiverValue)
	RecordMethodByName(methodValue2, "SetAge", receiverValue)

	entry1, found1 := GetCorrelation(receiverValue)
	if !found1 {
		t.Fatal("Expected first correlation to be found")
	}
	if entry1.MethodName != "SetAge" {
		t.Errorf("Expected most recent entry 'SetAge', got %s", entry1.MethodName)
	}

	entry2, found2 := GetCorrelation(receiverValue)
	if !found2 {
		t.Fatal("Expected second correlation to be found")
	}
	if entry2.MethodName != "GetName" {
		t.Errorf("Expected second entry 'GetName', got %s", entry2.MethodName)
	}

	if entry1.SequenceNum <= entry2.SequenceNum {
		t.Error("Expected first entry to have higher sequence number (most recent)")
	}

	_, found3 := GetCorrelation(receiverValue)
	if found3 {
		t.Error("Expected no more entries after consuming both")
	}
}

func TestRecordMethodByName_PrependKeepsLast10(t *testing.T) {
	resetTracker()

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)

	for i := 0; i < 15; i++ {
		methodValue := reflect.ValueOf(obj)
		RecordMethodByName(methodValue, "Method"+FormatUint64(uint64(i)), receiverValue)
	}

	val, ok := tracker.m.Load(ExtractValuePtr(receiverValue))
	if !ok {
		t.Fatal("Expected entries to exist")
	}

	entries := val.([]*Entry)
	if len(entries) != 10 {
		t.Errorf("Expected 10 entries (bounded), got %d", len(entries))
	}

	mostRecent := entries[0]
	if mostRecent.MethodName != "Method14" {
		t.Errorf("Expected most recent entry 'Method14', got %s", mostRecent.MethodName)
	}

	oldest := entries[9]
	if oldest.MethodName != "Method5" {
		t.Errorf("Expected oldest entry 'Method5', got %s", oldest.MethodName)
	}
}

func TestGetCorrelation_Consumption(t *testing.T) {
	resetTracker()

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)
	methodValue := reflect.ValueOf(obj)

	RecordMethodByName(methodValue, "GetName", receiverValue)

	entry1, found1 := GetCorrelation(receiverValue)
	if !found1 {
		t.Fatal("Expected correlation to be found")
	}

	entry2, found2 := GetCorrelation(receiverValue)
	if found2 {
		t.Error("Expected correlation to be consumed after first lookup")
	}

	if entry1.MethodName != "GetName" {
		t.Errorf("Expected method name 'GetName', got %s", entry1.MethodName)
	}

	if entry2 != nil {
		t.Error("Expected nil entry after consumption")
	}

	metrics := GetMetrics()
	if metrics["matches"] != 1 {
		t.Errorf("Expected 1 match, got %d", metrics["matches"])
	}
	if metrics["misses"] != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics["misses"])
	}
}

func TestGetCorrelationFromPtr(t *testing.T) {
	resetTracker()

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)
	methodValue := reflect.ValueOf(obj)

	RecordMethodByName(methodValue, "GetName", receiverValue)

	baton := ExtractValuePtr(receiverValue)
	entry, found := GetCorrelationFromPtr(baton)

	if !found {
		t.Fatal("Expected correlation to be found via GetCorrelationFromPtr")
	}

	if entry.MethodName != "GetName" {
		t.Errorf("Expected method name 'GetName', got %s", entry.MethodName)
	}

	if entry.ReceiverPtr != baton {
		t.Errorf("Expected receiverPtr to match baton")
	}
}

func TestGetCorrelation_ZeroBaton(t *testing.T) {
	resetTracker()

	var zeroValue reflect.Value
	entry, found := GetCorrelation(zeroValue)

	if found {
		t.Error("Expected no correlation for zero value")
	}

	if entry != nil {
		t.Error("Expected nil entry for zero value")
	}

	metrics := GetMetrics()
	if metrics["misses"] != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics["misses"])
	}
}

func TestRecordMethodByName_ZeroBaton(t *testing.T) {
	resetTracker()

	methodValue := reflect.ValueOf(&struct{}{})
	var zeroReceiver reflect.Value

	RecordMethodByName(methodValue, "GetName", zeroReceiver)

	baton := ExtractValuePtr(zeroReceiver)
	if baton != 0 {
		t.Error("Expected zero baton for zero receiver")
	}

	metrics := GetMetrics()
	if metrics["size"] != 0 {
		t.Errorf("Expected size 0, got %d", metrics["size"])
	}
}

func TestEvictLRU_WithSliceValues(t *testing.T) {
	resetTracker()

	os.Setenv(ENV_MAX_CORRELATIONS, "10")

	objs := make([]*struct{ name string }, 12)
	receiverValues := make([]reflect.Value, 12)

	for i := 0; i < 12; i++ {
		objs[i] = &struct{ name string }{name: FormatUint64(uint64(i))}
		receiverValues[i] = reflect.ValueOf(objs[i])
		methodValue := reflect.ValueOf(objs[i])
		RecordMethodByName(methodValue, "Method", receiverValues[i])
	}

	metrics := GetMetrics()
	if metrics["evictions"] == 0 {
		t.Error("Expected evictions to occur when exceeding maxEntries")
	}

	if metrics["size"] > 11 {
		t.Errorf("Expected size <= 11 (eviction happens before adding entry), got %d", metrics["size"])
	}
}

func TestCleanupByAge(t *testing.T) {
	resetTracker()

	os.Setenv(ENV_CORRELATION_MAX_AGE, "5")

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)
	methodValue := reflect.ValueOf(obj)

	RecordMethodByName(methodValue, "GetName", receiverValue)

	atomic.AddUint64(&tracker.sequence, 10)

	tracker.cleanupByAge()

	metrics := GetMetrics()
	if metrics["size"] != 0 {
		t.Errorf("Expected size 0 after cleanup, got %d", metrics["size"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	resetTracker()

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(methodNum int) {
			defer wg.Done()
			methodValue := reflect.ValueOf(obj)
			RecordMethodByName(methodValue, "Method"+FormatUint64(uint64(methodNum)), receiverValue)
		}(i)
	}

	wg.Wait()

	val, ok := tracker.m.Load(ExtractValuePtr(receiverValue))
	if !ok {
		t.Fatal("Expected entries to exist after concurrent recording")
	}

	entries := val.([]*Entry)
	if len(entries) != numGoroutines {
		t.Errorf("Expected %d entries, got %d", numGoroutines, len(entries))
	}

	metrics := GetMetrics()
	if metrics["size"] != 1 {
		t.Errorf("Expected size 1 (one receiver), got %d", metrics["size"])
	}
}

func TestMultipleReceivers(t *testing.T) {
	resetTracker()

	obj1 := &struct{ name string }{name: "obj1"}
	obj2 := &struct{ name string }{name: "obj2"}

	receiverValue1 := reflect.ValueOf(obj1)
	receiverValue2 := reflect.ValueOf(obj2)
	methodValue1 := reflect.ValueOf(obj1)
	methodValue2 := reflect.ValueOf(obj2)

	RecordMethodByName(methodValue1, "GetName1", receiverValue1)
	RecordMethodByName(methodValue2, "GetName2", receiverValue2)

	entry1, found1 := GetCorrelation(receiverValue1)
	if !found1 {
		t.Fatal("Expected correlation for obj1")
	}
	if entry1.MethodName != "GetName1" {
		t.Errorf("Expected 'GetName1', got %s", entry1.MethodName)
	}

	entry2, found2 := GetCorrelation(receiverValue2)
	if !found2 {
		t.Fatal("Expected correlation for obj2")
	}
	if entry2.MethodName != "GetName2" {
		t.Errorf("Expected 'GetName2', got %s", entry2.MethodName)
	}

	metrics := GetMetrics()
	if metrics["size"] != 0 {
		t.Errorf("Expected size 0 after consuming both, got %d", metrics["size"])
	}
}

func TestTemporalOrdering(t *testing.T) {
	resetTracker()

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)

	methods := []string{"Method1", "Method2", "Method3"}
	for _, methodName := range methods {
		methodValue := reflect.ValueOf(obj)
		RecordMethodByName(methodValue, methodName, receiverValue)
	}

	for i := len(methods) - 1; i >= 0; i-- {
		entry, found := GetCorrelation(receiverValue)
		if !found {
			t.Fatalf("Expected correlation for Method%d", i+1)
		}
		if entry.MethodName != methods[i] {
			t.Errorf("Expected %s, got %s (temporal ordering)", methods[i], entry.MethodName)
		}
	}
}

func resetTracker() {
	tracker = nil
	trackerOnce = sync.Once{}
	debugFile = nil
	debugOnce = sync.Once{}
	os.Unsetenv(ENV_MAX_CORRELATIONS)
	os.Unsetenv(ENV_CORRELATION_MAX_AGE)
	os.Unsetenv(ENV_CLEANUP_INTERVAL)
	os.Unsetenv(ENV_DEBUG_CORRELATION)
}

func TestValueOfWrapping(t *testing.T) {
	resetTracker()

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)
	methodValue := reflect.ValueOf(obj)

	RecordMethodByName(methodValue, "GetName", receiverValue)

	wrapped := reflect.ValueOf(methodValue)

	_, found := GetCorrelation(wrapped)
	if found {
		t.Error("Expected ValueOf wrapping to break direct correlation (uses fallback)")
	}

	receiverPtr := ExtractValuePtr(receiverValue)
	if receiverPtr != 0 {
		entry, found := GetCorrelationFromPtr(receiverPtr)
		if !found {
			t.Error("Expected fallback via receiver pointer to succeed")
		}
		if entry.MethodName != "GetName" {
			t.Errorf("Expected 'GetName', got %s", entry.MethodName)
		}
	}
}

func TestPointerVsValueReceiver(t *testing.T) {
	resetTracker()

	type TestStruct struct {
		Name string
	}

	ptrObj := &TestStruct{Name: "ptr"}
	valObj := TestStruct{Name: "val"}

	ptrReceiverValue := reflect.ValueOf(ptrObj)
	valReceiverValue := reflect.ValueOf(valObj)

	ptrMethodValue := reflect.ValueOf(ptrObj)
	valMethodValue := reflect.ValueOf(valObj)

	RecordMethodByName(ptrMethodValue, "GetName", ptrReceiverValue)
	RecordMethodByName(valMethodValue, "GetName", valReceiverValue)

	ptrEntry, ptrFound := GetCorrelation(ptrReceiverValue)
	if !ptrFound {
		t.Error("Expected correlation for pointer receiver")
	}
	if ptrEntry.MethodName != "GetName" {
		t.Errorf("Expected 'GetName', got %s", ptrEntry.MethodName)
	}

	valEntry, valFound := GetCorrelation(valReceiverValue)
	if !valFound {
		t.Error("Expected correlation for value receiver")
	}
	if valEntry.MethodName != "GetName" {
		t.Errorf("Expected 'GetName', got %s", valEntry.MethodName)
	}
}

func TestInterfaceConversion(t *testing.T) {
	resetTracker()

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)
	methodValue := reflect.ValueOf(obj)

	RecordMethodByName(methodValue, "GetName", receiverValue)

	var i interface{} = methodValue
	if m, ok := i.(reflect.Value); ok {
		entry, found := GetCorrelation(m)
		if !found {
			t.Error("Expected correlation to survive interface{} conversion with type assertion")
		}
		if entry.MethodName != "GetName" {
			t.Errorf("Expected 'GetName', got %s", entry.MethodName)
		}
	}

	i2 := (interface{})(methodValue)
	v2 := reflect.ValueOf(i2)
	if v2.Kind() == reflect.Func {
		entry, found := GetCorrelation(v2)
		if !found {
			t.Error("Expected correlation to survive ValueOf on interface{}")
		}
		if entry.MethodName != "GetName" {
			t.Errorf("Expected 'GetName', got %s", entry.MethodName)
		}
	}
}

func TestChainedMethods(t *testing.T) {
	resetTracker()

	type Nested struct {
		Inner *struct{ name string }
	}

	nested := &Nested{Inner: &struct{ name string }{name: "test"}}
	v := reflect.ValueOf(nested)
	elem := v.Elem()
	field := elem.FieldByName("Inner")
	innerValue := field.Elem()
	innerPtr := innerValue.Addr()

	RecordMethodByName(innerPtr, "GetName", innerPtr)

	entry, found := GetCorrelation(innerPtr)
	if !found {
		t.Error("Expected correlation to work after chained methods (Elem, Field, Addr)")
	}
	if entry.MethodName != "GetName" {
		t.Errorf("Expected 'GetName', got %s", entry.MethodName)
	}
}

func TestMakeFuncExpectedFailure(t *testing.T) {
	resetTracker()

	obj := &struct{ name string }{name: "test"}
	receiverValue := reflect.ValueOf(obj)
	methodValue := reflect.ValueOf(obj)

	RecordMethodByName(methodValue, "GetName", receiverValue)

	funcType := reflect.TypeOf(func() string { return "" })
	wrapper := reflect.MakeFunc(funcType, func(args []reflect.Value) []reflect.Value {
		return methodValue.Call(args)
	})

	entry, found := GetCorrelation(wrapper)
	if found {
		t.Error("Expected MakeFunc to break correlation (creates new function closure)")
	}
	if entry != nil {
		t.Error("Expected nil entry for MakeFunc")
	}
}
