// ABOUTME: GraphViz visualization package
// ABOUTME: Generates relationship, org chart, and pipeline graphs
package viz

import (
	"github.com/harperreed/pagen/repository"
)

type GraphGenerator struct {
	db *repository.DB
}

func NewGraphGenerator(db *repository.DB) *GraphGenerator {
	return &GraphGenerator{db: db}
}
