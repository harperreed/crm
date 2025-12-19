// ABOUTME: Visualization CLI commands
// ABOUTME: Handles viz dashboard and graph generation commands
package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/charm"
	"github.com/harperreed/pagen/viz"
)

// VizGraphContactsCommand generates a contact relationship network graph.
func VizGraphContactsCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("viz graph contacts", flag.ExitOnError)
	output := fs.String("output", "", "Output file (default: stdout)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	generator := viz.NewGraphGenerator(client)

	var contactID *uuid.UUID
	if fs.NArg() > 0 {
		id, err := uuid.Parse(fs.Arg(0))
		if err != nil {
			return fmt.Errorf("invalid contact ID: %w", err)
		}
		contactID = &id
	}

	dot, err := generator.GenerateContactGraph(contactID)
	if err != nil {
		return err
	}

	if *output != "" {
		return os.WriteFile(*output, []byte(dot), 0644)
	}

	fmt.Println(dot)
	return nil
}

// VizGraphCompanyCommand generates a company org chart.
func VizGraphCompanyCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("viz graph company", flag.ExitOnError)
	output := fs.String("output", "", "Output file (default: stdout)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("company ID required")
	}

	companyID, err := uuid.Parse(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid company ID: %w", err)
	}

	generator := viz.NewGraphGenerator(client)
	dot, err := generator.GenerateCompanyGraph(companyID)
	if err != nil {
		return err
	}

	if *output != "" {
		return os.WriteFile(*output, []byte(dot), 0644)
	}

	fmt.Println(dot)
	return nil
}

// VizGraphPipelineCommand generates a deal pipeline graph.
func VizGraphPipelineCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("viz graph pipeline", flag.ExitOnError)
	output := fs.String("output", "", "Output file (default: stdout)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	generator := viz.NewGraphGenerator(client)
	dot, err := generator.GeneratePipelineGraph()
	if err != nil {
		return err
	}

	if *output != "" {
		return os.WriteFile(*output, []byte(dot), 0644)
	}

	fmt.Println(dot)
	return nil
}

// VizGraphAllCommand generates a complete graph with all entities.
func VizGraphAllCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("viz graph all", flag.ExitOnError)
	output := fs.String("output", "", "Output file (default: stdout)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	generator := viz.NewGraphGenerator(client)
	dot, err := generator.GenerateCompleteGraph()
	if err != nil {
		return err
	}

	if *output != "" {
		return os.WriteFile(*output, []byte(dot), 0644)
	}

	fmt.Println(dot)
	return nil
}

func VizDashboardCommand(client *charm.Client, args []string) error {
	stats, err := viz.GenerateDashboardStats(client)
	if err != nil {
		return fmt.Errorf("failed to generate dashboard stats: %w", err)
	}

	output := viz.RenderDashboard(stats)
	fmt.Print(output)

	return nil
}
