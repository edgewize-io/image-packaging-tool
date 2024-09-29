package server

type ServerFile struct {
	Name        string           `yaml:"name"`
	Version     string           `yaml:"version"`
	Description string           `yaml:"description"`
	Servables   []ServableConfig `yaml:"servables"`
	Image       ImageInfo        `yaml:"image"`
}

type ServableConfig struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	ModelFile   string         `yaml:"modelFile,omitempty"`
	ModelFormat string         `yaml:"modelFormat,omitempty"`
	Methods     []MethodDetail `yaml:"methods"`
}

type MethodDetail struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Readme      string `yaml:"readme"`
}

type ImageInfo struct {
	Registry   string `yaml:"registry"`
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
}
