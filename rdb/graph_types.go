package rdb

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Vertex represents a graph node (vertex) parsed from agtype.
// Apache AGE returns vertices in the format: {"id": N, "label": "L", "properties": {...}}::vertex
type Vertex struct {
	ID         int64                  `json:"id"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties"`
}

// Edge represents a graph relationship parsed from agtype.
// Apache AGE returns edges in the format: {"id": N, "label": "L", "start_id": N, "end_id": N, "properties": {...}}::edge
type Edge struct {
	ID         int64                  `json:"id"`
	StartID    int64                  `json:"start_id"`
	EndID      int64                  `json:"end_id"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties"`
}

// Path represents a graph path (sequence of vertices and edges).
type Path struct {
	Vertices []Vertex `json:"vertices"`
	Edges    []Edge   `json:"edges"`
}

// agtypeTag constants for type suffix detection in agtype values.
const (
	agtypeVertexSuffix = "::vertex"
	agtypeEdgeSuffix   = "::edge"
	agtypePathSuffix   = "::path"
)

// ParseAgtype parses a raw agtype string returned by Apache AGE into
// a Go native type. The returned value can be one of:
//   - Vertex (when the value ends with ::vertex)
//   - Edge (when the value ends with ::edge)
//   - Path (when the value ends with ::path)
//   - map[string]interface{} (for generic JSON objects)
//   - a primitive (string, float64, bool, nil) for scalar values
func ParseAgtype(raw string) (interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	switch {
	case strings.HasSuffix(trimmed, agtypeVertexSuffix):
		return parseAgtypeVertex(trimmed)
	case strings.HasSuffix(trimmed, agtypeEdgeSuffix):
		return parseAgtypeEdge(trimmed)
	case strings.HasSuffix(trimmed, agtypePathSuffix):
		return parseAgtypePath(trimmed)
	default:
		return parseAgtypeScalar(trimmed)
	}
}

// parseAgtypeVertex parses a vertex agtype value such as:
// {"id": 1, "label": "Panel", "properties": {"sn": "ABC123"}}::vertex
func parseAgtypeVertex(raw string) (Vertex, error) {
	jsonStr := strings.TrimSuffix(raw, agtypeVertexSuffix)
	var v Vertex
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return Vertex{}, fmt.Errorf("%w: vertex: %v", ErrAgtypeParse, err)
	}
	return v, nil
}

// parseAgtypeEdge parses an edge agtype value such as:
// {"id": 2, "label": "RUNS_FIRMWARE", "start_id": 1, "end_id": 3, "properties": {}}::edge
func parseAgtypeEdge(raw string) (Edge, error) {
	jsonStr := strings.TrimSuffix(raw, agtypeEdgeSuffix)
	var e Edge
	if err := json.Unmarshal([]byte(jsonStr), &e); err != nil {
		return Edge{}, fmt.Errorf("%w: edge: %v", ErrAgtypeParse, err)
	}
	return e, nil
}

// parseAgtypePath parses a path agtype value. AGE returns paths as JSON arrays
// alternating between vertices and edges:
// [{"id":1,...}::vertex, {"id":2,...}::edge, {"id":3,...}::vertex]::path
func parseAgtypePath(raw string) (Path, error) {
	jsonStr := strings.TrimSuffix(raw, agtypePathSuffix)

	// AGE path format is a JSON array where elements alternate vertex, edge, vertex, ...
	// Each element in the array also has its own type suffix.
	// We need to strip the outer brackets and parse each element individually.
	jsonStr = strings.TrimSpace(jsonStr)
	if !strings.HasPrefix(jsonStr, "[") || !strings.HasSuffix(jsonStr, "]") {
		return Path{}, fmt.Errorf("%w: path: expected array format", ErrAgtypeParse)
	}

	// Remove outer brackets
	inner := jsonStr[1 : len(jsonStr)-1]
	elements := splitAgtypeElements(inner)

	path := Path{
		Vertices: make([]Vertex, 0),
		Edges:    make([]Edge, 0),
	}

	for _, elem := range elements {
		elem = strings.TrimSpace(elem)
		if elem == "" {
			continue
		}

		parsed, err := ParseAgtype(elem)
		if err != nil {
			return Path{}, fmt.Errorf("%w: path element: %v", ErrAgtypeParse, err)
		}

		switch v := parsed.(type) {
		case Vertex:
			path.Vertices = append(path.Vertices, v)
		case Edge:
			path.Edges = append(path.Edges, v)
		default:
			return Path{}, fmt.Errorf("%w: path: unexpected element type %T", ErrAgtypeParse, parsed)
		}
	}

	return path, nil
}

// parseAgtypeScalar parses a scalar agtype value (string, number, boolean, null, or generic JSON object).
func parseAgtypeScalar(raw string) (interface{}, error) {
	var result interface{}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		// If JSON parsing fails, return as a plain string (AGE sometimes returns unquoted strings).
		return raw, nil
	}
	return result, nil
}

// splitAgtypeElements splits a comma-separated list of agtype elements,
// respecting JSON nesting (braces and brackets) so that commas inside
// JSON objects or arrays are not treated as delimiters.
func splitAgtypeElements(s string) []string {
	var elements []string
	depth := 0
	start := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case ',':
			if depth == 0 {
				elements = append(elements, s[start:i])
				start = i + 1
			}
		}
	}

	// Add the last element
	if start < len(s) {
		elements = append(elements, s[start:])
	}

	return elements
}
