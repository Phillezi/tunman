package ssh

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/Phillezi/tunman-remaster/utils"
)

func GetHosts() []string {
	userHomeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(utils.Or(userHomeDir, os.Getenv("HOME")), ".ssh", "config")
	file, err := os.Open(os.ExpandEnv(configPath))
	if err != nil {
		return nil
	}
	defer file.Close()

	var hosts []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Host ") {
			fields := strings.Fields(line)
			hosts = append(hosts, fields[1:]...)
		}
	}
	return hosts
}
