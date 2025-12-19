// ABOUTME: GraphViz visualization package
// ABOUTME: Generates relationship, org chart, and pipeline graphs
package viz

import (
	"github.com/harperreed/pagen/charm"
)

type GraphGenerator struct {
	client *charm.Client
}

func NewGraphGenerator(client *charm.Client) *GraphGenerator {
	return &GraphGenerator{client: client}
}
