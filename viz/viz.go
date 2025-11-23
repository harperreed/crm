// ABOUTME: GraphViz visualization package
// ABOUTME: Generates relationship, org chart, and pipeline graphs
package viz

import (
	"database/sql"
)

type GraphGenerator struct {
	db *sql.DB
}

func NewGraphGenerator(db *sql.DB) *GraphGenerator {
	return &GraphGenerator{db: db}
}
