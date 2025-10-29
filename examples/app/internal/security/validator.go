package security

import "unsafe"

func ValidateInput(s string) bool {
	// Some validation logic
	return len(s) > 0
}

func UnsafeMemoryOperator(n int) bool {
	a := [16]int{3: 3, 9: 9, 11: 11, 15: n}
	eleSize := int(unsafe.Sizeof(a[0]))
	p9 := &a[9]
	up9 := unsafe.Pointer(p9)
	_ = (*int)(unsafe.Add(up9, -6*eleSize))
	return true
}
