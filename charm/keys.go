// ABOUTME: Key prefix constants for Charm KV storage
// ABOUTME: All keys follow the pattern "type:uuid" for prefix scanning

package charm

// Key prefixes for entity types
// Format: "prefix:uuid" enables efficient prefix scanning.
const (
	PrefixContact        = "contact:"
	PrefixCompany        = "company:"
	PrefixDeal           = "deal:"
	PrefixDealNote       = "dealnote:"
	PrefixRelationship   = "relationship:"
	PrefixInteractionLog = "interaction:"
	PrefixContactCadence = "cadence:"
	PrefixSuggestion     = "suggestion:"
	PrefixSyncState      = "syncstate:"
	PrefixSyncLog        = "synclog:"
)

// Key helper functions

// ContactKey returns the KV key for a contact.
func ContactKey(id string) []byte {
	return []byte(PrefixContact + id)
}

// CompanyKey returns the KV key for a company.
func CompanyKey(id string) []byte {
	return []byte(PrefixCompany + id)
}

// DealKey returns the KV key for a deal.
func DealKey(id string) []byte {
	return []byte(PrefixDeal + id)
}

// DealNoteKey returns the KV key for a deal note.
func DealNoteKey(id string) []byte {
	return []byte(PrefixDealNote + id)
}

// RelationshipKey returns the KV key for a relationship.
func RelationshipKey(id string) []byte {
	return []byte(PrefixRelationship + id)
}

// InteractionLogKey returns the KV key for an interaction log entry.
func InteractionLogKey(id string) []byte {
	return []byte(PrefixInteractionLog + id)
}

// ContactCadenceKey returns the KV key for a contact cadence
// Note: keyed by contact ID, not a separate cadence ID.
func ContactCadenceKey(contactID string) []byte {
	return []byte(PrefixContactCadence + contactID)
}

// SuggestionKey returns the KV key for a suggestion.
func SuggestionKey(id string) []byte {
	return []byte(PrefixSuggestion + id)
}

// SyncStateKey returns the KV key for sync state by service name.
func SyncStateKey(service string) []byte {
	return []byte(PrefixSyncState + service)
}

// SyncLogKey returns the KV key for a sync log entry.
func SyncLogKey(id string) []byte {
	return []byte(PrefixSyncLog + id)
}
