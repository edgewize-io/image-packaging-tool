package cmd

import (
	"fmt"
	localConfig "github.com/edgewize-io/image-packaging-tool/pkg/configuration"
	"github.com/edgewize-io/image-packaging-tool/pkg/utils"
	regConfig "github.com/regclient/regclient/config"
	"github.com/spf13/cobra"
	"os"
)

type LogoutOptions struct {
	rootOpts *RootOptions
}

func NewCmdLogout(rootOptions *RootOptions) *cobra.Command {
	logoutOptions := &LogoutOptions{
		rootOpts: rootOptions,
	}

	command := &cobra.Command{
		Use:   "logout <registry>",
		Short: "logout of a registry",
		Long:  `Remove registry credentials from the configuration.`,
		Example: `
# logout from Docker Hub
packctl logout

# logout from a specific registry
packctl logout registry.example.org`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return logoutOptions.runRegistryLogout(cmd, args)
		},
	}

	return command
}

func (out *LogoutOptions) runRegistryLogout(cmd *cobra.Command, args []string) (err error) {
	c, err := localConfig.ConfigLoadDefault()
	if err != nil {
		return err
	}
	if len(args) < 1 {
		args = []string{regConfig.DockerRegistry}
	}

	h := regConfig.HostNewName(args[0])
	if curH, ok := c.Hosts[h.Name]; ok {
		h = curH
	} else {
		utils.PrintWarning(os.Stdout, fmt.Sprintf("registry [%s] No configuration/credentials found\n", args[0]))
		return nil
	}
	h.User = ""
	h.Pass = ""
	h.Token = ""
	// TODO: add credHelper calls to erase a password
	err = c.ConfigSave()
	if err != nil {
		return
	}

	utils.PrintString(os.Stdout, fmt.Sprintf("registry [%s] logout\n", args[0]))
	return nil
}
