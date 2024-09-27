package cmd

import (
	"fmt"
	"github.com/edgewize-io/image-packaging-tool/pkg/constants"
	"github.com/edgewize-io/image-packaging-tool/pkg/utils"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

type InitOptions struct {
	description string
	version     string
}

func NewCmdInit() *cobra.Command {
	initOptions := &InitOptions{}
	command := &cobra.Command{
		Use:   "init",
		Short: "init metadata",
		Long:  "init metadata in current directory before making infer model images",
		RunE: func(cmd *cobra.Command, args []string) error {
			return initOptions.run()
		},
	}

	flags := command.Flags()
	flags.StringVar(&initOptions.description, "description", "", "new model description")
	flags.StringVar(&initOptions.version, "version", "latest", "new model version")
	return command
}

func (i *InitOptions) run() (err error) {
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	metaDirPath := filepath.Join(dir, constants.MetaDirName)
	err = os.Mkdir(metaDirPath, 0755)
	if os.IsExist(err) {
		utils.PrintYellow(os.Stdout, fmt.Sprintf("%s exists\n", metaDirPath))
		err = nil
	}

	if err != nil {
		return
	}

	serverConfigFilePath := filepath.Join(metaDirPath, constants.ServerConfigFile)
	_, err = os.Stat(serverConfigFilePath)
	if err == nil {
		utils.PrintYellow(os.Stdout, fmt.Sprintf("%s exist\n", serverConfigFilePath))
		return
	}

	if os.IsNotExist(err) {
		var file *os.File
		file, err = os.Create(serverConfigFilePath)
		if err != nil {
			return
		}

		defer file.Close()
	}

	return
}
