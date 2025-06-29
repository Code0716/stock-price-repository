package util

// ToPtrGenerics は与えられた値のポインタを返します。
// T は comparable インターフェースを満たす任意の型です。
//
// Example:
//
//	value := 42
//	ptr := ToPtrGenerics(value)
func ToPtrGenerics[T comparable](x T) *T {
	return &x
}

// FromPtrGenerics はポインタから値を取り出して返します。
// ポインタがnilの場合は、型Tのゼロ値を返します。
// T は comparable インターフェースを満たす任意の型です。
//
// Example:
//
//	ptr := ToPtrGenerics(42)
//	value := FromPtrGenerics(ptr)
func FromPtrGenerics[T comparable](x *T) T {
	if x == nil {
		var zero T
		return zero
	}
	return *x
}
