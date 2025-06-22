package main

import (
	"log"
	"os"

	"github.com/Phillezi/tunman-remaster/cmd/tunman/cli"
	"github.com/spf13/cobra/doc"
	"go.uber.org/zap"
)

const (
	docDir = "./docs"
)

func main() {
	if err := os.MkdirAll(docDir, 0755); err != nil {
		zap.L().Fatal("failed to create docs directory", zap.Error(err))
	}

	if err := doc.GenMarkdownTree(cli.GetRootCMD(), docDir); err != nil {
		zap.L().Fatal("failed to generate markdown docs", zap.Error(err))
	}
	log.Println("Docs generated successfully")
}
