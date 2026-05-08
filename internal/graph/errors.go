package graph

import "fmt"

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field string
}

func (e ConfigError) Error() string {
	return fmt.Sprintf("GRAPH_CONFIG_ERROR: missing or invalid field '%s'", e.Field)
}

// NewConfigError creates a new ConfigError
func NewConfigError(field string) ConfigError {
	return ConfigError{Field: field}
}

// ClientError represents a general client error
type ClientError struct {
	Message string
}

func (e ClientError) Error() string {
	return fmt.Sprintf("GRAPH_CLIENT_ERROR: %s", e.Message)
}

// NewClientError creates a new ClientError
func NewClientError(msg string) ClientError {
	return ClientError{Message: msg}
}
