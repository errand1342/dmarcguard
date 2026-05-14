package graph

import (
	"github.com/meysam81/parse-dmarc/internal/config"
)

// Config is an alias for the main config.GraphConfig type
// This allows the graph package to use the same config struct
type Config = config.GraphConfig
