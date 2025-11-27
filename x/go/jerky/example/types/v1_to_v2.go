package types

// MigrateUserV1ToV2Hook handles custom field mappings from V1 to V2.
// This file is generated once and will NOT be overwritten by jerky.
// New fields in V2: Role
func MigrateUserV1ToV2Hook(old *UserV1, new *UserV2) {
	// Add custom field mappings here. Examples:
	// new.NewField = old.OldField           // field rename
	// new.FullName = old.First + old.Last   // field merge
	// new.Role = "user"                     // default value for new field
}
