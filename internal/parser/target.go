package parser

import (
	"fmt"
	"strings"
)

// ParseTarget parses a string like "user@host:port" into its components.
func ParseTargetStrict(input string) (user, host, port string, err error) {
	// Split user and host:port
	parts := strings.Split(input, "@")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid format: expected user@host:port")
	}
	user = parts[0]

	// Split host and port
	hostPort := strings.Split(parts[1], ":")
	if len(hostPort) != 2 {
		return "", "", "", fmt.Errorf("invalid format: expected host:port")
	}
	host = hostPort[0]
	port = hostPort[1]

	return user, host, port, nil
}

// ParseTargetLoose attempts to parse input like "user@host:port" into components.
// It does its best to extract what's there, defaulting missing fields to empty strings.
func ParseTargetLoose(input string) (user, host, port string) {
	// Try splitting user and the rest
	atParts := strings.SplitN(input, "@", 2)
	if len(atParts) == 2 {
		user = atParts[0]
		input = atParts[1] // now input is "host[:port]"
	}

	// Split host and port if present
	colonParts := strings.SplitN(input, ":", 2)
	if len(colonParts) == 2 {
		host = colonParts[0]
		port = colonParts[1]
	} else {
		host = input
	}

	return user, host, port
}
