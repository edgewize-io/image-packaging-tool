package cmd

import (
	"fmt"
	"github.com/edgewize-io/image-packaging-tool/pkg/constants"
	"github.com/edgewize-io/image-packaging-tool/pkg/utils"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func NewCmdClean() *cobra.Command {
	command := &cobra.Command{
		Use:   "clean",
		Short: "clean current workspace metadata",
		Long:  "clean current workspace metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			return clean()
		},
	}

	return command
}

func clean() (err error) {
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	metaDirPath := filepath.Join(dir, constants.MetaDirName)
	err = os.RemoveAll(metaDirPath)
	if err == nil {
		utils.PrintString(os.Stdout, fmt.Sprintf("meta dir [%s] removed successfully\n", metaDirPath))
	}
	return
}
