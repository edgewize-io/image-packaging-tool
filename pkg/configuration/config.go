package configuration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/edgewize-io/image-packaging-tool/pkg/regctl/conffile"
	regConfig "github.com/regclient/regclient/config"
	"io"
	"io/fs"
)

var (
	ConfigFilename = "config.json"
	// ConfigDir is the default directory within the user's home directory to read/write configuration
	ConfigDir = ".packctl"
	// ConfigEnv is the environment variable to override the config filename
	ConfigEnv = "PACKCTL_CONFIG"
)

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

// ConfigSave saves to previously loaded filename
func (c *Config) ConfigSave() error {
	cf := conffile.New(conffile.WithFullname(c.Filename))
	if cf == nil {
		return fmt.Errorf("config not found")
	}
	out, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	outRdr := bytes.NewReader(out)
	return cf.Write(outRdr)
}
