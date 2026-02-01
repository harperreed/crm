package viz

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/harperreed/pagen/repository"
)

func (g *GraphGenerator) GeneratePipelineGraph() (string, error) {
	ctx := context.Background()
	gv, err := graphviz.New(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create graphviz instance: %w", err)
	}
	defer func() { _ = gv.Close() }()

	graph, err := gv.Graph()
	if err != nil {
		return "", fmt.Errorf("failed to create graph: %w", err)
	}
	defer func() { _ = graph.Close() }()

	graph.SetLayout("dot")
	graph.SetRankDir(cgraph.LRRank)

	// Get all deals
	deals, err := g.db.ListDeals(&repository.DealFilter{Limit: 10000})
	if err != nil {
		return "", fmt.Errorf("failed to fetch deals: %w", err)
	}

	// Group by stage
	stages := []string{
		repository.StageProspecting,
		repository.StageQualification,
		repository.StageProposal,
		repository.StageNegotiation,
		repository.StageClosedWon,
		repository.StageClosedLost,
	}

	dealsByStage := make(map[string][]*repository.Deal)
	for _, deal := range deals {
		stage := deal.Stage
		if stage == "" {
			stage = "unknown"
		}
		dealsByStage[stage] = append(dealsByStage[stage], deal)
	}

	// Create subgraphs for each stage
	for _, stage := range stages {
		if len(dealsByStage[stage]) == 0 {
			continue
		}

		subgraph, err := graph.CreateSubGraphByName(fmt.Sprintf("cluster_%s", stage))
		if err != nil {
			continue
		}
		subgraph.SetLabel(stage)

		for _, deal := range dealsByStage[stage] {
			label := fmt.Sprintf("%s\\n$%d", deal.Title, deal.Amount/100)
			node, err := subgraph.CreateNodeByName(label)
			if err != nil {
				continue
			}
			node.SetShape(cgraph.BoxShape)
		}
	}

	var buf bytes.Buffer
	if err := gv.Render(ctx, graph, graphviz.XDOT, &buf); err != nil {
		return "", fmt.Errorf("failed to render graph: %w", err)
	}

	return buf.String(), nil
}
