# Correlation Algorithm: Receiver Pointer Baton Pattern

## Overview

The algorithm uses the **receiver pointer** as a stable "baton" that survives all transformations to correlate dynamic operations. When static call graphs are incomplete due to dynamic dispatch (e.g., reflection, function pointers, or method lookups), this algorithm bridges the gap by tracking stable object identities across transformations. Currently applied to reflection operations (`MethodByName` → `Call`), but the pattern generalizes to any dynamic call pattern where object identity survives transformations.

**When to use:** This algorithm is specifically for two-phase dynamic operations where a lookup phase (e.g., `MethodByName("Encrypt")`) returns a value that's later used in an execution phase (e.g., `Call()`), creating a call graph gap that static analysis cannot bridge. Direct sequential operations (e.g., `hash.Write()` → `hash.Sum()`) don't need correlation since they maintain static connections.

## Mathematical Notation

### Definitions

- `obj ∈ T`: Original object instance (e.g., `*TestStruct`)
- `v = ValueOf(obj)`: Reflection Value wrapping the object
- `p = ptr(obj)`: Memory address (pointer) of the object
- `m = MethodByName(v, name)`: Method Value obtained via reflection
- `m' = ValueOf(m)`: Wrapped method Value (loses direct receiver access)
- `C`: Correlation map: `p → [(name₁, seq₁), (name₂, seq₂), ...]`

### Key Invariant

```
∀ transformations τ: ptr(τ(obj)) = ptr(obj)
```

**Note:** This invariant assumes the object is not moved by the garbage collector between `MethodByName` and `Call`. Go's GC is concurrent mark-and-sweep (not compacting), meaning objects are not moved in memory. Objects are not moved while actively referenced, and `reflect.Value` holds references that prevent GC movement. Correlation failures due to GC are theoretically possible but rare in practice due to the short correlation window (typically nanoseconds to milliseconds).

## Algorithm Flow

### Phase 1: Recording (MethodByName Time)

```
RECORD: p × name → C
  where p = ptr(receiverValue)
        C[p] = [(name, seq)] :: C[p]
```

**Process:**

1. Extract receiver pointer: `p = ptr(receiverValue)`
2. Create correlation entry: `(name, seq++)`
3. Store: `C[p] = [entry] + C[p]` (prepend, keep last 10)

### Phase 2: Lookup (Call Time)

```
LOOKUP: p → name | ""
  where p = ptr(callReceiverValue)

  if ∃ entry ∈ C[p]:
    return entry.name
    C[p] = C[p] \ {entry}  (consume on match)
  else:
    return ""
```

**Process:**

1. Extract receiver pointer: `p = ptr(callReceiverValue)`
2. Lookup: `C[p]`
3. If found: return most recent entry, consume from map
4. If not found: try fallback (extract from original receiver)

## Transformation Survival Matrix

| Transformation                          | Receiver Pointer Survives? | Correlation Possible? |
| --------------------------------------- | -------------------------- | --------------------- |
| Direct assignment (`m2 = m1`)           | Yes                        | Yes                   |
| Function parameter (`f(m)`)             | Yes                        | Yes                   |
| Struct storage (`{method: m}`)          | Yes                        | Yes                   |
| Interface conversion (`interface{}(m)`) | Yes                        | Yes                   |
| Type assertion (`i.(reflect.Value)`)    | Yes                        | Yes                   |
| Slice/Map storage                       | Yes                        | Yes                   |
| `ValueOf(m)` wrapping                   | No (method ptr changes)    | Fallback needed       |
| `MakeFunc` wrapper                      | No (new function)          | No                    |

**Notes:**

- Items 1-6: Very common patterns, fully supported
- `ValueOf(m)` wrapping: Rare, fallback mechanism handles it
- `MakeFunc`: Creates new functions - correlation failure is expected and indicates dynamic function creation

## Example Execution Traces

### Interface Conversion (Success)

```
p₀ = ptr(ValueOf(obj))           = 0x1000
C[p₀] = [("GetName", 1)]

τ₁ = interface{} conversion
τ₂ = ValueOf extraction

p₁ = ptr(τ₂(τ₁(ValueOf(obj))))  = 0x1000  // Invariant preserved!
LOOKUP(C, p₁) = "GetName"
```

### ValueOf Wrapping (Fallback)

```
p₀ = ptr(ValueOf(obj))           = 0x2000
C[p₀] = [("GetName", 2)]

wrapped = ValueOf(method)
p_wrapped = ptr(wrapped) = 0x3000  // Different pointer!

LOOKUP(C, p_wrapped) = Not found
FALLBACK: LOOKUP(C, p₀) = "GetName"  // Match found!
```

## Algorithm Properties

### Correctness

**Invariant:** If `m = MethodByName(v, name)` and `m.Call(...)` is invoked (possibly after transformations), the correlation lookup succeeds.

**Proof Sketch:**

1. At record time: `p = ptr(v)`, `C[p] = [(name, seq)]`
2. At lookup time: `p' = ptr(transformed_value)`
3. For transformations preserving receiver access: `p' = p` → lookup succeeds
4. For ValueOf wrapping: `p' ≠ p`, but `p` accessible via original receiver → fallback succeeds

### Performance

- **Space Complexity:** O(n) where n = number of unique receivers with active correlations
- **Time Complexity:**
  - Record: O(1) amortized (map insert + slice prepend)
  - Lookup: O(1) amortized (map lookup + slice head access)
  - LRU eviction: O(k) where k = number of methods per receiver (bounded by 10)

### Limitations

1. **MakeFunc:** Creates new functions → correlation not possible (expected behavior)
2. **ValueOf wrapping:** Requires fallback to original receiver pointer
3. **Multiple methods:** Temporal ordering assumption (most recent = most likely)
4. **Garbage Collection:** Go's GC is concurrent mark-and-sweep (not compacting), so objects are not moved in memory. However, there is a theoretical edge case: if an object becomes unreachable between `MethodByName` and `Call`, it could be collected. In practice:
   - Objects remain reachable through `reflect.Value` references
   - Correlation window is typically nanoseconds to milliseconds
   - GC runs asynchronously with stop-the-world pauses
   - Correlation failures due to GC are extremely rare and accepted as a known limitation
5. **Stack vs Heap:** Stack-allocated values passed to reflection typically escape to the heap via Go's escape analysis, making this rarely an issue in practice. Even if stack-allocated, the correlation window is short enough that stack frames remain valid.

### Concurrency Safety

The implementation uses `sync.Map` for thread-safe concurrent access without explicit mutexes. All map operations (`LoadOrStore`, `LoadAndDelete`, `CompareAndDelete`) are atomic and safe for concurrent use.

### Fallback Mechanism

For `ValueOf(m)` wrapping cases, the fallback can use `reflect.Value.Recv()` to extract the receiver from a method value:

```go
if callReceiverValue.Kind() == reflect.Func {
    if recv := callReceiverValue.Recv(); recv.IsValid() {
        p = ptr(recv)
        // Try lookup with receiver pointer
    }
}
```

## Implementation Pseudo-code

```go
type CorrelationKey struct {
    ReceiverPtr uintptr
}

type CorrelationEntry struct {
    MethodName  string
    SequenceNum uint64
}

// Record correlation
func record(receiverValue, methodValue, methodName) {
    p := ptr(receiverValue)  // Stable baton
    seq := atomic.Increment(sequence)
    C[p] = prepend(C[p], (methodName, seq))
    C[p] = take(C[p], 10)  // Keep last 10
}

// Lookup correlation
func lookup(callReceiverValue) (string, bool) {
    p := ptr(callReceiverValue)
    if entries := C[p]; entries != nil {
        entry := entries[0]  // Most recent
        C[p] = entries[1:]   // Consume
        return entry.name, true
    }

    // Fallback: if method value, extract receiver via Recv()
    if callReceiverValue.Kind() == reflect.Func {
        if recv := callReceiverValue.Recv(); recv.IsValid() {
            p = ptr(recv)
            if entries := C[p]; entries != nil {
                entry := entries[0]
                C[p] = entries[1:]
                return entry.name, true
            }
        }
    }

    return "", false
}
```

## Data Retention

Even when automatic correlation fails, complete log records are maintained:

```
Log: reflect.Value.MethodByName("GetName") called
Log: reflect.Value.Call() called (on method)
Log: reflect.MakeFunc() called (if applicable)
Log: reflect.Value.Call() called (on wrapper)
```

Manual analysis remains possible using temporal proximity, receiver pointers, call sites, and file/line information.

**Goal:** Maximize automatic correlation (handle most common cases) while ensuring all reflection operations are logged regardless of correlation success.
