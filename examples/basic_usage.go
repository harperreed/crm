// ABOUTME: Example demonstrating basic Office OS foundation API usage.
// ABOUTME: Shows common patterns for objects, relationships, and queries.

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/harperreed/pagen/db"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Open database
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// Initialize schema
	if err := db.InitSchema(database); err != nil {
		log.Fatal(err)
	}

	// Create repositories
	objRepo := db.NewObjectsRepository(database)
	relRepo := db.NewRelationshipsRepository(database)
	ctx := context.Background()

	fmt.Println("=== Office OS Foundation Example ===")

	// Example 1: Create objects
	fmt.Println("1. Creating objects...")

	company := &db.Object{
		Type: "Company",
		Name: "Tech Startup Inc",
		Metadata: map[string]interface{}{
			"domain":   "techstartup.io",
			"industry": "SaaS",
			"founded":  2024,
		},
	}
	if err := objRepo.Create(ctx, company); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created company: %s (ID: %s)\n", company.Name, company.ID)

	person := &db.Object{
		Type: "Person",
		Name: "Jane Developer",
		Metadata: map[string]interface{}{
			"email": "jane@techstartup.io",
			"role":  "CTO",
		},
	}
	if err := objRepo.Create(ctx, person); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created person: %s (ID: %s)\n\n", person.Name, person.ID)

	// Example 2: Create relationships
	fmt.Println("2. Creating relationships...")

	employment := &db.Relationship{
		SourceID: person.ID,
		TargetID: company.ID,
		Type:     "works_at",
		Metadata: map[string]interface{}{
			"start_date": "2024-01-01",
			"position":   "Chief Technology Officer",
		},
	}
	if err := relRepo.Create(ctx, employment); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created relationship: %s works_at %s\n\n", person.Name, company.Name)

	// Example 3: Query objects
	fmt.Println("3. Querying objects...")

	companies, err := objRepo.List(ctx, "Company")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d companies:\n", len(companies))
	for _, c := range companies {
		fmt.Printf("   - %s (domain: %s)\n", c.Name, c.Metadata["domain"])
	}
	fmt.Println()

	// Example 4: Query relationships
	fmt.Println("4. Querying relationships...")

	employments, err := relRepo.FindByTarget(ctx, company.ID, "works_at")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   %s has %d employees:\n", company.Name, len(employments))
	for _, emp := range employments {
		employee, err := objRepo.Get(ctx, emp.SourceID)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   - %s (%s)\n", employee.Name, emp.Metadata["position"])
	}
	fmt.Println()

	// Example 5: Update object metadata
	fmt.Println("5. Updating object metadata...")

	company.Metadata["employee_count"] = 10
	company.Metadata["funding_stage"] = "Seed"
	if err := objRepo.Update(ctx, company); err != nil {
		log.Fatal(err)
	}

	updated, err := objRepo.Get(ctx, company.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Updated %s:\n", updated.Name)
	fmt.Printf("   - Employees: %.0f\n", updated.Metadata["employee_count"])
	fmt.Printf("   - Funding: %s\n\n", updated.Metadata["funding_stage"])

	// Example 6: Create more complex relationships
	fmt.Println("6. Creating a project and task hierarchy...")

	project := &db.Object{
		Type: "Project",
		Name: "Mobile App Launch",
		Metadata: map[string]interface{}{
			"status":   "active",
			"priority": "high",
		},
	}
	if err := objRepo.Create(ctx, project); err != nil {
		log.Fatal(err)
	}

	task := &db.Object{
		Type: "Task",
		Name: "Design user interface",
		Metadata: map[string]interface{}{
			"status":     "in_progress",
			"assignee":   person.Name,
			"due_date":   "2025-01-15",
		},
	}
	if err := objRepo.Create(ctx, task); err != nil {
		log.Fatal(err)
	}

	// Task belongs to project
	taskToProject := &db.Relationship{
		SourceID: task.ID,
		TargetID: project.ID,
		Type:     "belongs_to",
	}
	if err := relRepo.Create(ctx, taskToProject); err != nil {
		log.Fatal(err)
	}

	// Person assigned to task
	assignment := &db.Relationship{
		SourceID: person.ID,
		TargetID: task.ID,
		Type:     "assigned_to",
		Metadata: map[string]interface{}{
			"assigned_date": "2025-01-01",
			"estimated_hours": 20,
		},
	}
	if err := relRepo.Create(ctx, assignment); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Created project: %s\n", project.Name)
	fmt.Printf("   Created task: %s\n", task.Name)
	fmt.Printf("   Assigned %s to task\n\n", person.Name)

	// Example 7: Complex queries
	fmt.Println("7. Complex queries...")

	// Find all tasks for a project
	projectTasks, err := relRepo.FindByTarget(ctx, project.ID, "belongs_to")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Project '%s' has %d tasks\n", project.Name, len(projectTasks))

	// Find what Jane is assigned to
	janeAssignments, err := relRepo.FindBySource(ctx, person.ID, "assigned_to")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   %s is assigned to %d tasks\n\n", person.Name, len(janeAssignments))

	// Example 8: List all objects by type
	fmt.Println("8. Summary of all objects:")
	allObjects, err := objRepo.List(ctx, "")
	if err != nil {
		log.Fatal(err)
	}

	typeCounts := make(map[string]int)
	for _, obj := range allObjects {
		typeCounts[obj.Type]++
	}

	for objType, count := range typeCounts {
		fmt.Printf("   - %s: %d\n", objType, count)
	}

	fmt.Println("\nExample completed successfully!")
}
