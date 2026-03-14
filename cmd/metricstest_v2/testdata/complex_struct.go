package testdata

// generate:reset
type NestedStruct struct {
	Field int
}

// generate:reset
type ComplexStruct struct {
	IntSlice  []int
	StringMap map[string]string
	Nested    *NestedStruct
	Pointer   *int
}
