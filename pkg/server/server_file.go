package server

type ServerFile struct {
	Name        string           `yaml:"name"`
	Version     string           `yaml:"version"`
	Description string           `yaml:"description"`
	Servables   []ServableConfig `yaml:"servables"`
}

type ServableConfig struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	ModelFile   string         `yaml:"modelFile,omitempty"`
	ModelFormat string         `yaml:"modelFormat,omitempty"`
	Methods     []MethodDetail `yaml:"methods"`
}

type MethodDetail struct {
	Name           string `yaml:"name"`
	Description    string `yaml:"description,omitempty"`
	ReadmeFilePath string `yaml:"readmeFilePath"`
}
