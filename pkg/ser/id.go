package ser

import (
	"encoding/base64"
	"errors"
	"strings"
)

// Ser combines tunnel and addrPair hashes into a combined id
func Ser(parentID, childID string) string {
	p := base64.RawURLEncoding.EncodeToString([]byte(parentID))
	c := base64.RawURLEncoding.EncodeToString([]byte(childID))
	return p + "." + c
}

// DeSer splits the combined ID back into tunnel and addrPair hashes
func DeSer(ser string) (string, string, error) {
	parts := strings.SplitN(ser, ".", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid serialized ID")
	}

	pBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", err
	}

	cBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", err
	}

	return string(pBytes), string(cBytes), nil
}
