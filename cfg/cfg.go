package cfg

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// Config defines the configuration for taskcluster-worker-starter.
type Config struct {
	// The worker-manager providerType (and, by implication, cloud) controlling
	// this worker.
	ProviderType string `yaml:"providerType"`

	// Configuration settings to be merged into the worker.
	WorkerConfig map[string]interface{} `yaml:"workerConfig"`
}

// Get a fragment of a usage message that describes the configuration file format
func Usage() string {
	return `
Configuration is in the form of a YAML file with the following fields:

	providerType: (required) the worker-manager providerType responsible for this worker;
		this generally indicates the cloud the worker is running in, or 'static' for a
		non-cloud-based worker

	workerConfig: arbitrary data which will be merged into the configuration given to
		the worker
`
}

// Load a configuration file
func Load(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
