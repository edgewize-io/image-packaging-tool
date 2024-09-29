package cmd

import (
	"fmt"
	"github.com/edgewize-io/image-packaging-tool/pkg/constants"
	"github.com/edgewize-io/image-packaging-tool/pkg/server"
	"github.com/edgewize-io/image-packaging-tool/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type InitOptions struct {
	description string
	version     string
	name        string
}

func NewCmdInit() *cobra.Command {
	initOptions := &InitOptions{}
	command := &cobra.Command{
		Use:   "init",
		Short: "init metadata",
		Long:  "init metadata in current directory before making infer model images",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				initOptions.name = args[0]
			} else {
				return fmt.Errorf("model service name cannot be empty")
			}

			return initOptions.run()
		},
	}

	flags := command.Flags()
	flags.StringVar(&initOptions.description, "name", "", "model service name")
	flags.StringVar(&initOptions.description, "description", "", "new model description")
	flags.StringVar(&initOptions.version, "version", "latest", "new model version")

	return command
}

func (ini *InitOptions) run() (err error) {
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
		workspaceName := ini.name
		if workspaceName == "" {
			workspaceName = filepath.Base(dir)
		}

		serverConfig := &server.ServerFile{
			Name:    workspaceName,
			Version: ini.version,
		}

		var serverConfigBytes []byte
		serverConfigBytes, err = yaml.Marshal(serverConfig)
		if err != nil {
			return
		}

		err = os.WriteFile(serverConfigFilePath, serverConfigBytes, 0666)
		if err != nil {
			return
		}

		utils.PrintString(os.Stdout, fmt.Sprintf("init current workspace successfully!\n"))
	}

	return
}
