package main

import (
	"github.com/edgewize-io/image-packaging-tool/cmd"
	"os"
)

func main() {
	rootCmd := cmd.NewImagePackagingCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
