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

type InitModelOptions struct {
	Name     string
	Override bool
}

func NewCmdInitModel() *cobra.Command {
	initModelOptions := &InitModelOptions{}
	command := &cobra.Command{
		Use:   "init-model",
		Short: "init new model",
		Long:  "init new model for current image",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				initModelOptions.Name = args[0]
			}

			if err := initModelOptions.Validate(); err != nil {
				return err
			}

			return initModelOptions.initModel()
		},
	}

	flags := command.Flags()
	flags.StringVar(&initModelOptions.Name, "name", "", "new model name")
	flags.BoolVar(&initModelOptions.Override, "override", true, "override existed servable.yaml file")
	return command
}

func (imo *InitModelOptions) Validate() (err error) {
	if imo.Name == "" {
		err = fmt.Errorf("name cannot be empty")
		return
	}

	return
}

func (imo *InitModelOptions) initModel() (err error) {
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	modelDir := filepath.Join(dir)
	err = os.Mkdir(modelDir, 0755)
	if os.IsExist(err) {
		utils.PrintYellow(os.Stdout, fmt.Sprintf("%s exists\n", modelDir))
		err = nil
	}

	if err != nil {
		return
	}

	servableFilePath := filepath.Join(modelDir, constants.ModelServableFile)
	_, err = os.Stat(servableFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			err = imo.initServableConfigFile(servableFilePath)
		}
		return
	}

	if imo.Override {
		err = imo.initServableConfigFile(servableFilePath)
	}

	return
}

func (imo *InitModelOptions) initServableConfigFile(filePath string) (err error) {
	servableConfig := &server.ServableConfig{
		Name:        imo.Name,
		Description: fmt.Sprintf("infer model %s", imo.Name),
		Methods:     make([]server.MethodDetail, 0),
	}

	var servableContentBytes []byte
	servableContentBytes, err = yaml.Marshal(servableConfig)
	if err != nil {
		return
	}

	err = os.WriteFile(filePath, servableContentBytes, 0666)
	if err == nil {
		utils.PrintString(os.Stdout, fmt.Sprintf("init servable.yaml [%s] successfully!\n", filePath))
	}
	return
}
