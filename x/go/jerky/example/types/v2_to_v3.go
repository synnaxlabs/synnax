package types

// MigrateUserV2ToV3Hook handles custom field mappings from V2 to V3.
// This file is generated once and will NOT be overwritten by jerky.
// New fields in V3: Score, Verified
func MigrateUserV2ToV3Hook(old *UserV2, new *UserV3) {
	// Add custom field mappings here. Examples:
	// new.NewField = old.OldField           // field rename
	// new.FullName = old.First + old.Last   // field merge
	// new.Role = "user"                     // default value for new field
}
