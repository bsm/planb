package planb

import "github.com/hashicorp/raft"

// Config contains server config directives
type Config struct {
	// Raft configuration options
	Raft *raft.Config

	// Sentinel configuration
	Sentinel struct {
		// MasterName must be set to enable sentinel support
		MasterName string
	}
}

// NewConfig inits a default configuration
func NewConfig() *Config {
	return &Config{
		Raft: raft.DefaultConfig(),
	}
}

func (c *Config) norm(fn string) error {
	if c.Raft == nil {
		c.Raft = raft.DefaultConfig()
	}
	return normNodeID(c.Raft, fn)
}
