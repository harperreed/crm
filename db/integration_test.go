// ABOUTME: Integration tests demonstrating Office OS foundation in action.
// ABOUTME: Shows real-world scenarios combining objects and relationships.

package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCRMScenario demonstrates a basic CRM setup with people, companies, and relationships.
func TestCRMScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	// Create a company
	acme := &Object{
		Type: "Company",
		Name: "Acme Corporation",
		Metadata: map[string]interface{}{
			"domain":   "acme.com",
			"industry": "Technology",
			"size":     "500-1000",
		},
	}
	require.NoError(t, objRepo.Create(ctx, acme))

	// Create people
	alice := &Object{
		Type: "Person",
		Name: "Alice Johnson",
		Metadata: map[string]interface{}{
			"email": "alice@acme.com",
			"title": "VP of Engineering",
		},
	}
	require.NoError(t, objRepo.Create(ctx, alice))

	bob := &Object{
		Type: "Person",
		Name: "Bob Smith",
		Metadata: map[string]interface{}{
			"email": "bob@acme.com",
			"title": "Senior Engineer",
		},
	}
	require.NoError(t, objRepo.Create(ctx, bob))

	// Create relationships
	aliceWorksAt := &Relationship{
		SourceID: alice.ID,
		TargetID: acme.ID,
		Type:     "works_at",
		Metadata: map[string]interface{}{
			"start_date": "2020-01-15",
			"department": "Engineering",
		},
	}
	require.NoError(t, relRepo.Create(ctx, aliceWorksAt))

	bobWorksAt := &Relationship{
		SourceID: bob.ID,
		TargetID: acme.ID,
		Type:     "works_at",
		Metadata: map[string]interface{}{
			"start_date": "2021-06-01",
			"department": "Engineering",
		},
	}
	require.NoError(t, relRepo.Create(ctx, bobWorksAt))

	aliceManagesBob := &Relationship{
		SourceID: alice.ID,
		TargetID: bob.ID,
		Type:     "manages",
	}
	require.NoError(t, relRepo.Create(ctx, aliceManagesBob))

	// Query: Find all people at Acme
	employeesRels, err := relRepo.FindByTarget(ctx, acme.ID, "works_at")
	require.NoError(t, err)
	assert.Len(t, employeesRels, 2, "Should have 2 employees")

	// Query: Find who Alice manages
	managedRels, err := relRepo.FindBySource(ctx, alice.ID, "manages")
	require.NoError(t, err)
	assert.Len(t, managedRels, 1, "Alice should manage 1 person")
	assert.Equal(t, bob.ID, managedRels[0].TargetID)

	// Query: Find Alice's company
	aliceCompanyRels, err := relRepo.FindBySource(ctx, alice.ID, "works_at")
	require.NoError(t, err)
	require.Len(t, aliceCompanyRels, 1)

	aliceCompany, err := objRepo.Get(ctx, aliceCompanyRels[0].TargetID)
	require.NoError(t, err)
	assert.Equal(t, "Acme Corporation", aliceCompany.Name)

	// Test cascade: Delete Acme should remove work relationships
	require.NoError(t, objRepo.Delete(ctx, acme.ID))

	remainingRels, err := relRepo.List(ctx, "works_at")
	require.NoError(t, err)
	assert.Empty(t, remainingRels, "Work relationships should cascade delete")

	managementRels, err := relRepo.List(ctx, "manages")
	require.NoError(t, err)
	assert.Len(t, managementRels, 1, "Management relationships should remain")
}

// TestProjectManagementScenario demonstrates a project management setup.
func TestProjectManagementScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	// Create a project
	project := &Object{
		Type: "Project",
		Name: "Website Redesign",
		Metadata: map[string]interface{}{
			"status":     "active",
			"priority":   "high",
			"start_date": "2025-01-01",
			"budget":     100000,
		},
	}
	require.NoError(t, objRepo.Create(ctx, project))

	// Create tasks
	task1 := &Object{
		Type: "Task",
		Name: "Design mockups",
		Metadata: map[string]interface{}{
			"status":       "completed",
			"estimated_hours": 20,
		},
	}
	require.NoError(t, objRepo.Create(ctx, task1))

	task2 := &Object{
		Type: "Task",
		Name: "Implement frontend",
		Metadata: map[string]interface{}{
			"status":       "in_progress",
			"estimated_hours": 40,
		},
	}
	require.NoError(t, objRepo.Create(ctx, task2))

	// Create person
	developer := &Object{
		Type: "Person",
		Name: "Carol Developer",
		Metadata: map[string]interface{}{
			"email": "carol@example.com",
		},
	}
	require.NoError(t, objRepo.Create(ctx, developer))

	// Link tasks to project
	rel1 := &Relationship{
		SourceID: task1.ID,
		TargetID: project.ID,
		Type:     "belongs_to",
	}
	require.NoError(t, relRepo.Create(ctx, rel1))

	rel2 := &Relationship{
		SourceID: task2.ID,
		TargetID: project.ID,
		Type:     "belongs_to",
	}
	require.NoError(t, relRepo.Create(ctx, rel2))

	// Assign developer to task
	assignment := &Relationship{
		SourceID: developer.ID,
		TargetID: task2.ID,
		Type:     "assigned_to",
		Metadata: map[string]interface{}{
			"role": "lead",
		},
	}
	require.NoError(t, relRepo.Create(ctx, assignment))

	// Query: Get all tasks for project
	projectTasks, err := relRepo.FindByTarget(ctx, project.ID, "belongs_to")
	require.NoError(t, err)
	assert.Len(t, projectTasks, 2, "Project should have 2 tasks")

	// Query: Get developer's assignments
	assignments, err := relRepo.FindBySource(ctx, developer.ID, "assigned_to")
	require.NoError(t, err)
	assert.Len(t, assignments, 1)

	assignedTask, err := objRepo.Get(ctx, assignments[0].TargetID)
	require.NoError(t, err)
	assert.Equal(t, "Implement frontend", assignedTask.Name)
	assert.Equal(t, "in_progress", assignedTask.Metadata["status"])
}

// TestKnowledgeGraphScenario demonstrates a knowledge graph use case.
func TestKnowledgeGraphScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	// Create concepts
	ai := &Object{
		Type: "Concept",
		Name: "Artificial Intelligence",
		Metadata: map[string]interface{}{
			"definition": "The simulation of human intelligence in machines",
		},
	}
	require.NoError(t, objRepo.Create(ctx, ai))

	ml := &Object{
		Type: "Concept",
		Name: "Machine Learning",
		Metadata: map[string]interface{}{
			"definition": "A subset of AI that enables learning from data",
		},
	}
	require.NoError(t, objRepo.Create(ctx, ml))

	deepLearning := &Object{
		Type: "Concept",
		Name: "Deep Learning",
		Metadata: map[string]interface{}{
			"definition": "ML using neural networks with multiple layers",
		},
	}
	require.NoError(t, objRepo.Create(ctx, deepLearning))

	// Create hierarchical relationships
	mlPartOfAI := &Relationship{
		SourceID: ml.ID,
		TargetID: ai.ID,
		Type:     "is_part_of",
	}
	require.NoError(t, relRepo.Create(ctx, mlPartOfAI))

	dlPartOfML := &Relationship{
		SourceID: deepLearning.ID,
		TargetID: ml.ID,
		Type:     "is_part_of",
	}
	require.NoError(t, relRepo.Create(ctx, dlPartOfML))

	// Create a document that references these concepts
	paper := &Object{
		Type: "Document",
		Name: "Introduction to Neural Networks",
		Metadata: map[string]interface{}{
			"author": "Dr. Smith",
			"year":   2024,
		},
	}
	require.NoError(t, objRepo.Create(ctx, paper))

	// Link document to concepts
	paperMentionsAI := &Relationship{
		SourceID: paper.ID,
		TargetID: ai.ID,
		Type:     "mentions",
	}
	require.NoError(t, relRepo.Create(ctx, paperMentionsAI))

	paperMentionsDL := &Relationship{
		SourceID: paper.ID,
		TargetID: deepLearning.ID,
		Type:     "mentions",
	}
	require.NoError(t, relRepo.Create(ctx, paperMentionsDL))

	// Query: What concepts does the paper mention?
	mentionedConcepts, err := relRepo.FindBySource(ctx, paper.ID, "mentions")
	require.NoError(t, err)
	assert.Len(t, mentionedConcepts, 2, "Paper should mention 2 concepts")

	// Query: What are the sub-concepts of AI?
	aiSubconcepts, err := relRepo.FindByTarget(ctx, ai.ID, "is_part_of")
	require.NoError(t, err)
	assert.Len(t, aiSubconcepts, 1)

	subConcept, err := objRepo.Get(ctx, aiSubconcepts[0].SourceID)
	require.NoError(t, err)
	assert.Equal(t, "Machine Learning", subConcept.Name)

	// Query: What documents mention Deep Learning?
	dlMentions, err := relRepo.FindByTarget(ctx, deepLearning.ID, "mentions")
	require.NoError(t, err)
	assert.Len(t, dlMentions, 1)
}

// TestDynamicMetadataEvolution shows how metadata can evolve over time.
func TestDynamicMetadataEvolution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	ctx := context.Background()

	// Create a product with minimal metadata
	product := &Object{
		Type: "Product",
		Name: "Smart Thermostat",
		Metadata: map[string]interface{}{
			"sku": "THERM-001",
		},
	}
	require.NoError(t, objRepo.Create(ctx, product))

	// Later, add more metadata
	product.Metadata["price"] = 199.99
	product.Metadata["stock"] = 50
	product.Metadata["features"] = []interface{}{
		"WiFi enabled",
		"Voice control",
		"Energy saving",
	}
	require.NoError(t, objRepo.Update(ctx, product))

	// Retrieve and verify
	retrieved, err := objRepo.Get(ctx, product.ID)
	require.NoError(t, err)

	assert.Equal(t, "THERM-001", retrieved.Metadata["sku"])
	assert.Equal(t, 199.99, retrieved.Metadata["price"])
	assert.Equal(t, float64(50), retrieved.Metadata["stock"])

	features := retrieved.Metadata["features"].([]interface{})
	assert.Len(t, features, 3)
	assert.Equal(t, "WiFi enabled", features[0])
}

// TestMultiTypeRelationships demonstrates different relationship types between same objects.
func TestMultiTypeRelationships(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	alice := &Object{Type: "Person", Name: "Alice"}
	bob := &Object{Type: "Person", Name: "Bob"}

	require.NoError(t, objRepo.Create(ctx, alice))
	require.NoError(t, objRepo.Create(ctx, bob))

	// Multiple relationship types between the same people
	manages := &Relationship{
		SourceID: alice.ID,
		TargetID: bob.ID,
		Type:     "manages",
	}
	require.NoError(t, relRepo.Create(ctx, manages))

	mentors := &Relationship{
		SourceID: alice.ID,
		TargetID: bob.ID,
		Type:     "mentors",
	}
	require.NoError(t, relRepo.Create(ctx, mentors))

	collaborates := &Relationship{
		SourceID: alice.ID,
		TargetID: bob.ID,
		Type:     "collaborates_with",
	}
	require.NoError(t, relRepo.Create(ctx, collaborates))

	// Query all relationships between Alice and Bob
	allRels, err := relRepo.FindBetween(ctx, alice.ID, bob.ID)
	require.NoError(t, err)
	assert.Len(t, allRels, 3, "Should have 3 different relationship types")

	// Verify we have all three types
	types := make(map[string]bool)
	for _, rel := range allRels {
		types[rel.Type] = true
	}

	assert.True(t, types["manages"])
	assert.True(t, types["mentors"])
	assert.True(t, types["collaborates_with"])
}
