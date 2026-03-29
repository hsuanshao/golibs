package rdb

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/hsuanshao/golibs/ctx"
)

// GraphClient provides Apache AGE (Cypher) query capabilities
// on top of an existing DBClient PostgreSQL connection.
type GraphClient interface {
	// ExecuteCypher runs a Cypher query and returns parsed results.
	// Results are returned as a slice of maps, where each map is a row,
	// and keys are the Cypher RETURN aliases.
	ExecuteCypher(ctx ctx.CTX, graphName string, cypher string, args ...interface{}) ([]map[string]interface{}, error)

	// MutateCypher runs a Cypher mutation (CREATE/MERGE/SET/DELETE).
	// Returns the number of affected rows.
	MutateCypher(ctx ctx.CTX, graphName string, cypher string, args ...interface{}) (int64, error)
}

// graphClientImpl is the concrete implementation of GraphClient.
type graphClientImpl struct {
	db DBClient
}

type GraphDBClient DBClient

// NewGraphClient creates a new GraphClient from an existing DBClient.
// The provided DBClient determines which pool (Leader/Follower) is used.
// Typical usage: NewGraphClient(dbClient.Follower()) for read queries on the analytics node.
func NewGraphClient(db GraphDBClient) GraphClient {
	return &graphClientImpl{db: db}
}

// buildCypherSQL wraps a Cypher query string into the Apache AGE SQL format.
// The returnAliases parameter specifies the column aliases in the AS clause;
// each alias is typed as agtype. If returnAliases is empty, a single "result" alias is used.
func buildCypherSQL(graphName string, cypher string, returnAliases []string) string {
	if len(returnAliases) == 0 {
		returnAliases = []string{"result"}
	}

	var asCols []string
	for _, alias := range returnAliases {
		asCols = append(asCols, fmt.Sprintf("%s agtype", alias))
	}

	return fmt.Sprintf(
		"SELECT * FROM cypher('%s', $$ %s $$) AS (%s)",
		graphName,
		cypher,
		strings.Join(asCols, ", "),
	)
}

// extractReturnAliases parses the RETURN clause of a Cypher query to extract
// the column aliases. It handles basic cases like:
//
//	RETURN n, f           → ["n", "f"]
//	RETURN n AS node      → ["node"]
//	RETURN n.name, count  → ["n.name", "count"]
//
// For mutations without RETURN, it returns an empty slice.
func extractReturnAliases(cypher string) []string {
	upper := strings.ToUpper(cypher)
	idx := strings.LastIndex(upper, "RETURN ")
	if idx == -1 {
		return nil
	}

	returnClause := strings.TrimSpace(cypher[idx+7:])
	parts := strings.Split(returnClause, ",")

	var aliases []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for AS aliasing (case-insensitive)
		upperPart := strings.ToUpper(part)
		asIdx := strings.LastIndex(upperPart, " AS ")
		if asIdx != -1 {
			alias := strings.TrimSpace(part[asIdx+4:])
			aliases = append(aliases, alias)
		} else {
			// Use the expression itself as the alias
			aliases = append(aliases, part)
		}
	}

	return aliases
}

// ExecuteCypher runs a Cypher query and returns parsed results.
// Results are returned as a slice of maps, where each map represents a row.
// Map keys are the Cypher RETURN aliases, and values are parsed agtype values
// (Vertex, Edge, Path, or primitive types).
func (g *graphClientImpl) ExecuteCypher(c ctx.CTX, graphName string, cypher string, args ...interface{}) ([]map[string]interface{}, error) {
	if graphName == "" {
		return nil, &GraphError{Op: "ExecuteCypher", Graph: graphName, Message: ErrGraphNameEmpty.Error(), Err: ErrGraphNameEmpty}
	}
	if cypher == "" {
		return nil, &GraphError{Op: "ExecuteCypher", Graph: graphName, Message: ErrCypherEmpty.Error(), Err: ErrCypherEmpty}
	}

	aliases := extractReturnAliases(cypher)
	sql := buildCypherSQL(graphName, cypher, aliases)

	// Use the aliases for map keys; if none extracted, default to ["result"]
	if len(aliases) == 0 {
		aliases = []string{"result"}
	}

	c.WithFields(logrus.Fields{
		"graph":  graphName,
		"cypher": cypher,
		"sql":    sql,
	}).Debug("executing cypher query")

	rows, dbErr := g.db.Query(c, sql, args...)
	if dbErr != nil {
		return nil, &GraphError{Op: "ExecuteCypher", Graph: graphName, Message: "query execution failed", Err: dbErr}
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		// Scan all columns as raw strings (agtype is text-based)
		rawValues := make([]interface{}, len(aliases))
		scanDests := make([]interface{}, len(aliases))
		for i := range rawValues {
			scanDests[i] = &rawValues[i]
		}

		if err := rows.Scan(scanDests...); err != nil {
			return nil, &GraphError{Op: "ExecuteCypher", Graph: graphName, Message: "scan failed", Err: err}
		}

		row := make(map[string]interface{}, len(aliases))
		for i, alias := range aliases {
			rawStr, ok := rawValues[i].(string)
			if !ok {
				// If value is nil or not a string, store as-is
				row[alias] = rawValues[i]
				continue
			}

			parsed, err := ParseAgtype(rawStr)
			if err != nil {
				return nil, &GraphError{Op: "ExecuteCypher", Graph: graphName, Message: fmt.Sprintf("agtype parse failed for column %q", alias), Err: err}
			}
			row[alias] = parsed
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, &GraphError{Op: "ExecuteCypher", Graph: graphName, Message: "rows iteration error", Err: err}
	}

	return results, nil
}

// MutateCypher runs a Cypher mutation (CREATE/MERGE/SET/DELETE).
// It returns the number of rows affected by the mutation.
func (g *graphClientImpl) MutateCypher(c ctx.CTX, graphName string, cypher string, args ...interface{}) (int64, error) {
	if graphName == "" {
		return 0, &GraphError{Op: "MutateCypher", Graph: graphName, Message: ErrGraphNameEmpty.Error(), Err: ErrGraphNameEmpty}
	}
	if cypher == "" {
		return 0, &GraphError{Op: "MutateCypher", Graph: graphName, Message: ErrCypherEmpty.Error(), Err: ErrCypherEmpty}
	}

	aliases := extractReturnAliases(cypher)
	sql := buildCypherSQL(graphName, cypher, aliases)

	c.WithFields(logrus.Fields{
		"graph":  graphName,
		"cypher": cypher,
		"sql":    sql,
	}).Debug("executing cypher mutation")

	tag, dbErr := g.db.Exec(c, sql, args...)
	if dbErr != nil {
		return 0, &GraphError{Op: "MutateCypher", Graph: graphName, Message: "mutation execution failed", Err: dbErr}
	}

	return tag.RowsAffected(), nil
}
