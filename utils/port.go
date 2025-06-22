package utils

import (
	"fmt"
	"strconv"
)

// ParsePortStrict safely parses a port string into a uint32.
// Returns an error if the value is not a valid port (0–65535).
func ParsePortStrict(s string) (uint32, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: %w", s, err)
	}
	if i < 0 || i > 65535 {
		return 0, fmt.Errorf("port out of range: %d", i)
	}
	return uint32(i), nil
}

func ParsePort(s string) uint32 {
	v, _ := ParsePortStrict(s)
	return v
}
