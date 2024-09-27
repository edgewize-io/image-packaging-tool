package cmd

import (
	"github.com/spf13/cobra"
)

func NewImagePackagingCommand() *cobra.Command {
	packCtlCmd := &cobra.Command{
		Use:  "packctl",
		Long: "cli tool for packaging inference model images",
	}

	packCtlCmd.AddCommand(NewCmdInit())
	packCtlCmd.AddCommand(NewCmdInitModel())

	setFlags(packCtlCmd)

	return packCtlCmd
}

func setFlags(cmd *cobra.Command) {
	//flags := cmd.PersistentFlags()
	//flags.Bool("create", false, "Create a new kubeconfig file if not exists")
}
