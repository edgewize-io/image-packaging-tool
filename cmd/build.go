package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/edgewize-io/image-packaging-tool/pkg/regctl/conffile"
	"github.com/edgewize-io/image-packaging-tool/pkg/regctl/strparse"
	"github.com/edgewize-io/image-packaging-tool/pkg/regctl/version"
	"github.com/edgewize-io/image-packaging-tool/pkg/utils"
	"github.com/regclient/regclient"
	regConfig "github.com/regclient/regclient/config"
	"github.com/regclient/regclient/mod"
	"github.com/regclient/regclient/pkg/archive"
	"github.com/regclient/regclient/scheme/reg"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"
)

var (
	// ConfigFilename is the default filename to read/write configuration
	ConfigFilename = "config.json"
	// ConfigDir is the default directory within the user's home directory to read/write configuration
	ConfigDir = ".regctl"
	// ConfigEnv is the environment variable to override the config filename
	ConfigEnv = "REGCTL_CONFIG"

	UserAgent = "imagePackingTool/packctl"
)

type BuildOptions struct {
	BaseImage   string
	TargetImage string
	Platforms   string

	name      string
	verbosity string
	logopts   []string
	format    string // for Go template formatting of various commands
	hosts     []string
	userAgent string
}

type Config struct {
	Filename      string                     `json:"-"`                 // filename that was loaded
	Version       int                        `json:"version,omitempty"` // version the file in case the config file syntax changes in the future
	Hosts         map[string]*regConfig.Host `json:"hosts,omitempty"`
	HostDefault   *regConfig.Host            `json:"hostDefault,omitempty"`
	BlobLimit     int64                      `json:"blobLimit,omitempty"`
	IncDockerCert *bool                      `json:"incDockerCert,omitempty"`
	IncDockerCred *bool                      `json:"incDockerCred,omitempty"`
}

// ConfigNew creates an empty configuration
func ConfigNew() *Config {
	c := Config{
		Hosts: map[string]*regConfig.Host{},
	}
	return &c
}

// ConfigLoadConfFile loads the config from an io reader
func ConfigLoadConfFile(cf *conffile.File) (*Config, error) {
	r, err := cf.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	c := ConfigNew()
	if err := json.NewDecoder(r).Decode(c); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	c.Filename = cf.Name()
	// verify loaded version is not higher than supported version
	if c.Version > 1 {
		return c, fmt.Errorf("unsupported config version")
	}
	for h := range c.Hosts {
		if c.Hosts[h].Name == "" {
			c.Hosts[h].Name = h
		}
		if c.Hosts[h].Hostname == "" {
			c.Hosts[h].Hostname = h
		}
		if c.Hosts[h].TLS == regConfig.TLSUndefined {
			c.Hosts[h].TLS = regConfig.TLSEnabled
		}
		if h == regConfig.DockerRegistryDNS || h == regConfig.DockerRegistry || h == regConfig.DockerRegistryAuth {
			// Docker Hub
			c.Hosts[h].Name = regConfig.DockerRegistry
			if c.Hosts[h].Hostname == h {
				c.Hosts[h].Hostname = regConfig.DockerRegistryDNS
			}
			if c.Hosts[h].CredHost == h {
				c.Hosts[h].CredHost = regConfig.DockerRegistryAuth
			}
		}
		// ensure key matches Name
		if c.Hosts[h].Name != h {
			c.Hosts[c.Hosts[h].Name] = c.Hosts[h]
			delete(c.Hosts, h)
		}
	}
	return c, nil
}

// ConfigLoadDefault loads the config from the (default) filename
func ConfigLoadDefault() (*Config, error) {
	cf := conffile.New(conffile.WithDirName(ConfigDir, ConfigFilename), conffile.WithEnvFile(ConfigEnv))
	if cf == nil {
		return nil, fmt.Errorf("failed to define config file")
	}
	c, err := ConfigLoadConfFile(cf)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		// do not error on file not found
		c := ConfigNew()
		c.Filename = cf.Name()
		return c, nil
	}
	return c, err
}

func NewCmdBuild() *cobra.Command {
	buildOptions := &BuildOptions{}
	command := &cobra.Command{
		Use:   "build",
		Short: "build model image",
		Long:  "init new model for current image",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				buildOptions.TargetImage = args[0]
			}

			return buildOptions.build()
		},
	}

	flags := command.Flags()
	flags.StringVar(&buildOptions.BaseImage, "baseImage", "", "base infer model image")
	flags.StringVar(&buildOptions.Platforms, "platforms", "linux/amd64", "build image platforms, default: linux/amd64")
	command.PersistentFlags().StringVarP(&buildOptions.verbosity, "verbosity", "v", logrus.WarnLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")
	command.PersistentFlags().StringArrayVar(&buildOptions.logopts, "logopt", []string{}, "Log options")
	command.PersistentFlags().StringArrayVar(&buildOptions.hosts, "host", []string{}, "Registry hosts to add (reg=registry,user=username,pass=password,tls=enabled)")
	command.PersistentFlags().StringVarP(&buildOptions.userAgent, "user-agent", "", "", "Override user agent")
	return command
}

func (bo *BuildOptions) build() (err error) {
	ctx := context.TODO()
	currWorkDir, err := os.Getwd()
	if err != nil {
		return
	}

	var rdr io.Reader
	var platforms []platform.Platform

	pf, err := platform.Parse(bo.Platforms)
	if err != nil {
		err = fmt.Errorf("failed to parse platform %s: %v", bo.Platforms, err)
		return
	}

	platforms = append(platforms, pf)

	pr, pw := io.Pipe()
	go func() {
		err := archive.Tar(context.TODO(), currWorkDir, pw)
		if err != nil {
			_ = pw.CloseWithError(err)
		}
		_ = pw.Close()
	}()
	rdr = pr
	defer pr.Close()

	rSrc, err := ref.New(bo.BaseImage)
	if err != nil {
		return
	}

	var rTgt ref.Ref
	if strings.ContainsAny(bo.TargetImage, "/:") {
		rTgt, err = ref.New(bo.TargetImage)
		if err != nil {
			err = fmt.Errorf("failed to parse new image name %s: %w", bo.TargetImage, err)
			return
		}
	} else {
		rTgt = rSrc.SetTag(bo.TargetImage)
	}

	modOptions := []mod.Opts{}
	modOptions = append(modOptions, mod.WithLayerAddTar(rdr, "", platforms), mod.WithRefTgt(rTgt))

	rc := bo.newRegClient()
	defer rc.Close(ctx, rSrc)
	rOut, err := mod.Apply(ctx, rc, rSrc, modOptions...)
	if err != nil {
		return
	}

	utils.PrintString(os.Stdout, fmt.Sprintf("%s\n", rOut.CommonName()))
	err = rc.Close(ctx, rOut)
	if err != nil {
		err = fmt.Errorf("failed to close ref: %w", err)
	}
	return
}

func (bo *BuildOptions) newRegClient() *regclient.RegClient {
	conf, err := ConfigLoadDefault()
	if err != nil {
		utils.PrintWarning(os.Stdout, fmt.Sprintf("failed to load default\n"))
		if conf == nil {
			conf = ConfigNew()
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
	if bo.userAgent != "" {
		rcOpts = append(rcOpts, regclient.WithUserAgent(bo.userAgent))
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
	for _, h := range bo.hosts {
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
