// ABOUTME: Relationship CLI commands
// ABOUTME: Human-friendly commands for managing contact relationships
package cli

import (
	"flag"
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/charm"
)

// UpdateRelationshipCommand updates a relationship.
func UpdateRelationshipCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("update-relationship", flag.ExitOnError)
	relType := fs.String("type", "", "Relationship type")
	context := fs.String("context", "", "Relationship context")
	_ = fs.Parse(args)

	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: update-relationship <id> [--type <type>] [--context <context>]")
	}

	relID, err := uuid.Parse(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid relationship ID: %w", err)
	}

	rel, err := client.GetRelationship(relID)
	if err != nil {
		return err
	}

	if *relType != "" {
		rel.RelationshipType = *relType
	}
	if *context != "" {
		rel.Context = *context
	}

	err = client.UpdateRelationship(rel)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Updated relationship: %s\n", relID)
	return nil
}

// DeleteRelationshipCommand deletes a relationship.
func DeleteRelationshipCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("delete-relationship", flag.ExitOnError)
	_ = fs.Parse(args)

	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: delete-relationship <id>")
	}

	relID, err := uuid.Parse(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid relationship ID: %w", err)
	}

	err = client.DeleteRelationship(relID)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Deleted relationship: %s\n", relID)
	return nil
}
