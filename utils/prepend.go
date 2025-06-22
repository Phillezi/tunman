package utils

func Prepend[T any](original []T, elems ...T) []T {
	return append(elems, original...)
}
