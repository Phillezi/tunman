package utils

func Or[T comparable](vals ...T) T {
	var zero T
	for _, val := range vals {
		if val != zero {
			return val
		}
	}
	// all are zero values
	return zero
}
