// ABOUTME: Tests for base Office OS object types
// ABOUTME: Validates BaseObject structure and JSON serialization
package objects

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseObject_JSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	id := uuid.New()
	createdBy := uuid.New()

	obj := BaseObject{
		ID:        id,
		Kind:      "task",
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		ACL: []ACLEntry{
			{ActorID: createdBy, Role: "owner"},
		},
		Tags:   []string{"urgent", "crm"},
		Fields: map[string]interface{}{"title": "Test task"},
	}

	// Test serialization
	data, err := json.Marshal(obj)
	require.NoError(t, err)

	// Test deserialization
	var decoded BaseObject
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, obj.ID, decoded.ID)
	assert.Equal(t, obj.Kind, decoded.Kind)
	assert.Equal(t, obj.CreatedAt.Unix(), decoded.CreatedAt.Unix())
	assert.Equal(t, obj.UpdatedAt.Unix(), decoded.UpdatedAt.Unix())
	assert.Equal(t, obj.CreatedBy, decoded.CreatedBy)
	assert.Equal(t, len(obj.ACL), len(decoded.ACL))
	assert.Equal(t, obj.ACL[0].ActorID, decoded.ACL[0].ActorID)
	assert.Equal(t, obj.ACL[0].Role, decoded.ACL[0].Role)
	assert.Equal(t, obj.Tags, decoded.Tags)
}

func TestACLEntry_JSONSerialization(t *testing.T) {
	actorID := uuid.New()
	acl := ACLEntry{
		ActorID: actorID,
		Role:    "owner",
	}

	data, err := json.Marshal(acl)
	require.NoError(t, err)

	var decoded ACLEntry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, acl.ActorID, decoded.ActorID)
	assert.Equal(t, acl.Role, decoded.Role)
}
