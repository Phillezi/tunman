package parser

import (
	"fmt"
	"strings"

	"github.com/Phillezi/tunman/internal/defaults"
	"github.com/Phillezi/tunman/utils"
)

func ParsePublishes(publishes []string) (map[string]string, error) {
	if len(publishes) == 0 {
		return nil, nil
	}
	localRemoteMap := make(map[string]string)

	for _, publish := range publishes {
		parts := strings.SplitN(publish, ":", 4)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid publish format, (less than 2 parts)")
		}
		switch len(parts) {
		case 2:
			localRemoteMap[fmt.Sprintf("%s:%s", defaults.DefaultPublishHost, parts[0])] = fmt.Sprintf("%s:%s", defaults.DefaultPublishHost, parts[1])
		case 3:
			_, err := utils.ParsePortStrict(parts[0])
			if err != nil {
				// the first two are local and the third is remote
				localRemoteMap[fmt.Sprintf("%s:%s", parts[0], parts[1])] = fmt.Sprintf("%s:%s", defaults.DefaultPublishHost, parts[2])
			} else {
				// the first is local (the port) and the second two are remote
				localRemoteMap[fmt.Sprintf("%s:%s", defaults.DefaultPublishHost, parts[0])] = fmt.Sprintf("%s:%s", parts[1], parts[2])
			}
		case 4:
			localRemoteMap[fmt.Sprintf("%s:%s", parts[0], parts[1])] = fmt.Sprintf("%s:%s", parts[2], parts[3])
		}
	}

	return localRemoteMap, nil
}
