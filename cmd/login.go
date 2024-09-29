package cmd

import (
	"bufio"
	"fmt"
	localConfig "github.com/edgewize-io/image-packaging-tool/pkg/configuration"
	"github.com/edgewize-io/image-packaging-tool/pkg/imageref"
	"github.com/edgewize-io/image-packaging-tool/pkg/utils"
	regConfig "github.com/regclient/regclient/config"
	"github.com/regclient/regclient/types/ref"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type LoginOptions struct {
	rootOpts  *RootOptions
	registry  string
	user      string
	pass      string // login opts
	passStdin bool
	skipCheck bool
}

func NewCmdLogin(rootOptions *RootOptions) *cobra.Command {
	loginOptions := &LoginOptions{
		rootOpts: rootOptions,
	}

	command := &cobra.Command{
		Use:   "login <registry>",
		Short: "login to a registry",
		Long: `Provide login credentials for a registry. This may not be necessary if you
have already logged in with docker.`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return loginOptions.runRegistryLogin(cmd, args)
		},
	}

	command.Flags().StringVarP(&loginOptions.user, "user", "u", "", "Username")
	command.Flags().StringVarP(&loginOptions.pass, "pass", "p", "", "Password")
	command.Flags().BoolVar(&loginOptions.passStdin, "pass-stdin", false, "Read password from stdin")
	command.Flags().BoolVar(&loginOptions.skipCheck, "skip-check", false, "Skip checking connectivity to the registry")

	return command
}

func (lo *LoginOptions) runRegistryLogin(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()
	// disable signal handler to allow ctrl-c to be used on prompts (context cancel on a blocking reader is difficult)
	signal.Reset(os.Interrupt, syscall.SIGTERM)
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
		c.Hosts[h.Name] = h
	}

	if flagChanged(cmd, "user") {
		h.User = lo.user
	} else if lo.passStdin {
		err = fmt.Errorf("user must be provided to read password from stdin")
		return
	} else {
		// prompt for username
		reader := bufio.NewReader(cmd.InOrStdin())
		defUser := ""
		if h.User != "" {
			defUser = " [" + h.User + "]"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Enter Username%s: ", defUser)
		user, _ := reader.ReadString('\n')
		user = strings.TrimSpace(user)
		if user != "" {
			h.User = user
		} else if h.User == "" {
			err = fmt.Errorf("Username is required")
			return
		}
	}

	var pass []byte
	if flagChanged(cmd, "pass") {
		h.Pass = lo.pass
	} else if lo.passStdin {
		pass, err = io.ReadAll(cmd.InOrStdin())
		if err != nil {
			err = fmt.Errorf("failed to read password from stdin: %w", err)
			return
		}
		passwd := strings.TrimRight(string(pass), "\n")
		if passwd != "" {
			h.Pass = passwd
		} else {
			err = fmt.Errorf("Password is required")
			return
		}
	} else {
		// prompt for a password
		var fd int
		if ifd, ok := cmd.InOrStdin().(interface{ Fd() uintptr }); ok {
			fd = int(ifd.Fd())
		} else {
			return fmt.Errorf("file descriptor needed to prompt for password (resolve by using \"-p\" flag)")
		}

		fmt.Fprint(cmd.OutOrStdout(), "Enter Password: ")
		pass, err = term.ReadPassword(fd)
		if err != nil {
			err = fmt.Errorf("unable to read from tty (resolve by using \"-p\" flag, or winpty on Windows): %w", err)
			return
		}

		passwd := strings.TrimRight(string(pass), "\n")
		fmt.Fprint(cmd.OutOrStdout(), "\n")
		if passwd != "" {
			h.Pass = passwd
		} else {
			err = fmt.Errorf("Password is required")
			return
		}
	}

	err = c.ConfigSave()
	if err != nil {
		return
	}

	if !lo.skipCheck {
		var imageRef imageref.ImageRef
		imageRef, err = imageref.NewHost(args[0])
		if err != nil {
			return
		}
		rc := lo.rootOpts.newRegClient()
		_, err = rc.Ping(ctx, ref.Ref{
			Scheme:   "reg",
			Registry: imageRef.Registry,
		})
		if err != nil {
			utils.PrintWarning(os.Stdout, fmt.Sprintf("Failed to ping registry, credentials were still stored\n"))
			return err
		}
	}

	utils.PrintString(os.Stdout, fmt.Sprintf("login registry [%s] successfully!\n", args[0]))
	return nil
}

func flagChanged(cmd *cobra.Command, name string) bool {
	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		return false
	}
	return flag.Changed
}
