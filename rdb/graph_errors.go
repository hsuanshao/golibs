package rdb

import (
	"errors"
	"fmt"
)

// GraphError represents an error that occurred during a graph operation.
// It preserves the original error chain for debugging while providing
// a descriptive message for the caller.
type GraphError struct {
	Op      string // operation that failed, e.g. "ExecuteCypher"
	Graph   string // graph name involved
	Message string // human-readable description
	Err     error  // underlying error
}

// Error implements the error interface for GraphError.
func (e *GraphError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("graph(%s) %s: %s: %v", e.Graph, e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("graph(%s) %s: %s", e.Graph, e.Op, e.Message)
}

// Unwrap returns the underlying error for errors.Is / errors.As support.
func (e *GraphError) Unwrap() error {
	return e.Err
}

// Sentinel errors for graph operations.
var (
	// ErrGraphNameEmpty is returned when an empty graph name is provided.
	ErrGraphNameEmpty = errors.New("graph name must not be empty")

	// ErrCypherEmpty is returned when an empty Cypher query string is provided.
	ErrCypherEmpty = errors.New("cypher query must not be empty")

	// ErrAgtypeParse is returned when an agtype value cannot be parsed.
	ErrAgtypeParse = errors.New("failed to parse agtype value")

	// ErrScanDestination is returned when the scan destination is invalid.
	ErrScanDestination = errors.New("scan destination must be a non-nil pointer to a slice of structs")

	// ErrGraphQueryFailed is returned when the underlying SQL query execution fails.
	ErrGraphQueryFailed = errors.New("graph query execution failed")

	// ErrGraphMutateFailed is returned when a graph mutation operation fails.
	ErrGraphMutateFailed = errors.New("graph mutation execution failed")
)
