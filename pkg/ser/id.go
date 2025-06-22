package ser

import (
	"errors"
	"strings"
)

// Ser combines tunnel and addrPair hashes into a combined id
func Ser(parentID, childID string) string {
	return parentID + "." + childID
}

// DeSer splits the combined ID back into tunnel and addrPair hashes
func DeSer(ser string) (string, string, error) {
	parts := strings.SplitN(ser, ".", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid serialized ID")
	}

	return parts[0], parts[1], nil
}
