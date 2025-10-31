# Call Graph Reconstruction with Correlation Tracking

This document demonstrates how correlation tracking bridges call graph gaps created by reflection.

## Minimal Example

### Source Code

```go
package main

import (
	"fmt"
	"reflect"
)

type Calculator struct {
	result int
}

func (c *Calculator) Add(x int) {
	c.result += x
}

func (c *Calculator) GetResult() int {
	return c.result
}

func main() {
	calc := &Calculator{result: 10}

	val := reflect.ValueOf(calc)

	method := val.MethodByName("Add")
	method.Call([]reflect.Value{reflect.ValueOf(5)})

	method = val.MethodByName("GetResult")
	result := method.Call([]reflect.Value{})

	if len(result) > 0 {
		fmt.Printf("Result: %d\n", result[0].Int())
	}
}
```

## The Call Graph Gap

Static analysis tools generate a call graph showing:

```
main.main calls reflect.Value.MethodByName
main.main calls reflect.Value.Call
```

**The problem:** The static call graph cannot determine:

- Which method name was passed to `MethodByName()`
- What `Call()` actually invokes at runtime
- Whether `MethodByName` and `Call` are related

## Correlation Tracking Solution

The instrumentation logs capture the missing information:

**Log entries:**

```json
{"operation":"reflect.Value.MethodByName","v":"1374390599712","name":"Add"}
{"operation":"reflect.Value.Call","v":"1374390599712","method_name":"Add","correlation_seq":"1"}
{"operation":"reflect.Value.MethodByName","v":"1374390599712","name":"GetResult"}
{"operation":"reflect.Value.Call","v":"1374390599712","method_name":"GetResult","correlation_seq":"2"}
```

**Key fields:**

- `v`: Receiver pointer (stable identifier) = `1374390599712`
- `name`: Method name from `MethodByName()`
- `method_name`: Method name logged in `Call()`
- `correlation_seq`: Sequence number for temporal ordering

## Reconstruction Process

Match operations using the receiver pointer (`v`) and correlation sequence:

**Correlation 1:**

```
MethodByName("Add") [v=1374390599712]
  → Call() [v=1374390599712, method_name="Add", seq=1]
    → Calculator.Add()
```

**Correlation 2:**

```
MethodByName("GetResult") [v=1374390599712]
  → Call() [v=1374390599712, method_name="GetResult", seq=2]
    → Calculator.GetResult()
```

## Complete Reconstructed Call Graph

**Before (Static Only):**

```
main.main
  └─> reflect.Value.MethodByName [unknown method]
  └─> reflect.Value.Call [unknown invocation]
  └─> reflect.Value.MethodByName [unknown method]
  └─> reflect.Value.Call [unknown invocation]
```

**After (With Correlation):**

```
main.main
  └─> reflect.Value.MethodByName("Add")
  └─> reflect.Value.Call() [method_name="Add", seq=1]
      └─> Calculator.Add()
  └─> reflect.Value.MethodByName("GetResult")
  └─> reflect.Value.Call() [method_name="GetResult", seq=2]
      └─> Calculator.GetResult()
```

## How It Works

1. **Recording:** When `MethodByName()` is called, record the method name with the receiver pointer
2. **Matching:** When `Call()` is invoked, lookup using the receiver pointer
3. **Bridging:** The correlation data connects `MethodByName` to `Call`, revealing the actual method invoked

The receiver pointer (`v`) serves as a stable identifier that survives transformations between `MethodByName` and `Call`, enabling complete call graph reconstruction.

## Analysis Results

Running the analysis script shows:

```
MethodByName Operations:
  v=1374390599712 → MethodByName("Add")
  v=1374390599712 → MethodByName("GetResult")

Call Operations:
  v=1374390599712 → Call() [method_name="Add", seq=1]
  v=1374390599712 → Call() [method_name="GetResult", seq=2]

Matched Correlations:
  MethodByName("Add") → Call() → Calculator.Add()
  MethodByName("GetResult") → Call() → Calculator.GetResult()

Successfully matched: 2/2
```

Both correlations matched successfully, demonstrating that the receiver pointer (`v=1374390599712`) correctly bridges the gap between `MethodByName` and `Call` operations.
