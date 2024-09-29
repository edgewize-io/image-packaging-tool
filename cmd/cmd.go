package cmd

import (
	"fmt"
	localConfig "github.com/edgewize-io/image-packaging-tool/pkg/configuration"
	"github.com/edgewize-io/image-packaging-tool/pkg/regctl/strparse"
	"github.com/edgewize-io/image-packaging-tool/pkg/regctl/version"
	"github.com/edgewize-io/image-packaging-tool/pkg/utils"
	"github.com/regclient/regclient"
	regConfig "github.com/regclient/regclient/config"
	"github.com/regclient/regclient/scheme/reg"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var UserAgent = "imagePackingTool/packctl"

type RootOptions struct {
	name      string
	verbosity string
	logopts   []string
	format    string // for Go template formatting of various commands
	hosts     []string
	userAgent string
}

func NewImagePackagingCommand() *cobra.Command {
	rootOptions := &RootOptions{}
	packCtlCmd := &cobra.Command{
		Use:  "packctl",
		Long: "cli tool for packaging inference model images",
	}

	packCtlCmd.AddCommand(NewCmdInit())
	packCtlCmd.AddCommand(NewCmdClean())
	packCtlCmd.AddCommand(NewCmdBuild(rootOptions))
	packCtlCmd.AddCommand(NewCmdLogin(rootOptions))
	packCtlCmd.AddCommand(NewCmdLogout(rootOptions))

	packCtlCmd.PersistentFlags().StringVarP(&rootOptions.verbosity, "verbosity", "v", logrus.WarnLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")
	packCtlCmd.PersistentFlags().StringArrayVar(&rootOptions.logopts, "logopt", []string{}, "Log options")
	packCtlCmd.PersistentFlags().StringArrayVar(&rootOptions.hosts, "host", []string{}, "Registry hosts to add (reg=registry,user=username,pass=password,tls=enabled)")
	packCtlCmd.PersistentFlags().StringVarP(&rootOptions.userAgent, "user-agent", "", "", "Override user agent")

	return packCtlCmd
}

func (ro *RootOptions) newRegClient() *regclient.RegClient {
	conf, err := localConfig.ConfigLoadDefault()
	if err != nil {
		utils.PrintWarning(os.Stdout, fmt.Sprintf("failed to load default\n"))
		if conf == nil {
			conf = localConfig.ConfigNew()
		}
	}

	regLog := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.WarnLevel,
	}

	rcOpts := []regclient.Opt{
		regclient.WithLog(regLog),
		regclient.WithRegOpts(reg.WithCache(time.Minute*5, 500)),
	}
	if ro.userAgent != "" {
		rcOpts = append(rcOpts, regclient.WithUserAgent(ro.userAgent))
	} else {
		info := version.GetInfo()
		if info.VCSTag != "" {
			rcOpts = append(rcOpts, regclient.WithUserAgent(UserAgent+" ("+info.VCSTag+")"))
		} else {
			rcOpts = append(rcOpts, regclient.WithUserAgent(UserAgent+" ("+info.VCSRef+")"))
		}
	}
	if conf.BlobLimit != 0 {
		rcOpts = append(rcOpts, regclient.WithRegOpts(reg.WithBlobLimit(conf.BlobLimit)))
	}
	if conf.IncDockerCred == nil || *conf.IncDockerCred {
		rcOpts = append(rcOpts, regclient.WithDockerCreds())
	}
	if conf.IncDockerCert == nil || *conf.IncDockerCert {
		rcOpts = append(rcOpts, regclient.WithDockerCerts())
	}

	rcHosts := []regConfig.Host{}
	for name, host := range conf.Hosts {
		host.Name = name
		rcHosts = append(rcHosts, *host)
	}
	for _, h := range ro.hosts {
		hKV, err := strparse.SplitCSKV(h)
		if err != nil {
			utils.PrintWarning(os.Stdout, fmt.Sprintf("unable to parse host string\n"))
		}
		host := regConfig.Host{
			Name: hKV["reg"],
			User: hKV["user"],
			Pass: hKV["pass"],
		}
		if hKV["tls"] != "" {
			var hostTLS regConfig.TLSConf
			err := hostTLS.UnmarshalText([]byte(hKV["tls"]))
			if err != nil {
				utils.PrintWarning(os.Stdout, fmt.Sprintf("unable to parse tls setting\n"))
			} else {
				host.TLS = hostTLS
			}
		}
		rcHosts = append(rcHosts, host)
	}
	if len(rcHosts) > 0 {
		rcOpts = append(rcOpts, regclient.WithConfigHost(rcHosts...))
	}

	return regclient.New(rcOpts...)
}
