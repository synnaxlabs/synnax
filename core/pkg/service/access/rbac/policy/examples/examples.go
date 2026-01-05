// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package examples contains example policies demonstrating the constraint system.
// These are for documentation and testing purposes only.
package examples

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/x/telem"
)

// =============================================================================
// CONSTRAINT TYPES:
//
// 1. FieldConstraint - Direct field access on resources, subjects, requests, system
//    Example: resource.status == "active", subject.clearance in ["secret", "top_secret"]
//
// 2. RelationshipConstraint - Ontology graph traversal
//    Example: resource created_by subject, resource labeled_by "production"
//
// 3. ComputedConstraint - Derived/calculated values
//    Example: request.time_range duration <= 24h
// =============================================================================

// =============================================================================
// ORIGINAL EXAMPLES
// =============================================================================

// CreatorOnlySchematicUpdate allows users to only update schematics they created.
// Uses RelationshipConstraint to check the "created_by" ontology relationship.
var CreatorOnlySchematicUpdate = policy.Policy{
	Key:     uuid.New(),
	Name:    "creator-only-schematic-update",
	Objects: []ontology.ID{{Type: "schematic"}},
	Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Relationship{
			Relationship: "created_by",
			Operator:     constraint.RelOpContainsSome,
			MatchSubject: true,
		},
	},
}

// TimeRangeChannelRead restricts channel data access to a specific time range.
// Uses FieldConstraint to check the request's time_range field.
func TimeRangeChannelRead(channelKey string, allowedRange telem.TimeRange) policy.Policy {
	return policy.Policy{
		Key:     uuid.New(),
		Name:    "time-range-channel-read",
		Objects: []ontology.ID{{Type: "channel", Key: channelKey}},
		Actions: []access.Action{access.ActionRetrieve},
		Effect:  policy.EffectAllow,
		Constraints: []constraint.Constraint{
			constraint.Field{
				Target:   "request",
				Field:    []string{"time_range"},
				Operator: constraint.OpWithin,
				Value:    allowedRange,
			},
		},
	}
}

// PropertyRestrictedStatusUpdate allows updating only specific properties on a status.
// Uses FieldConstraint to check the request's properties field.
var PropertyRestrictedStatusUpdate = policy.Policy{
	Key:     uuid.New(),
	Name:    "property-restricted-status-update",
	Objects: []ontology.ID{{Type: "status"}},
	Actions: []access.Action{access.ActionUpdate},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "request",
			Field:    []string{"properties"},
			Operator: constraint.OpSubsetOf,
			Value:    []string{"details", "variant"},
		},
	},
}

// ErrorStatusesOnly restricts retrieval to statuses in error state.
// Uses FieldConstraint to check the resource's status field.
var ErrorStatusesOnly = policy.Policy{
	Key:     uuid.New(),
	Name:    "error-statuses-only",
	Objects: []ontology.ID{{Type: "status"}},
	Actions: []access.Action{access.ActionRetrieve},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "resource",
			Field:    []string{"status"},
			Operator: constraint.OpEqual,
			Value:    "error",
		},
	},
}

// OPCScanTasksOnly restricts task retrieval to opc_scan type tasks.
// Uses FieldConstraint to check the resource's type field.
var OPCScanTasksOnly = policy.Policy{
	Key:     uuid.New(),
	Name:    "opc-scan-tasks-only",
	Objects: []ontology.ID{{Type: "task"}},
	Actions: []access.Action{access.ActionRetrieve},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "resource",
			Field:    []string{"type"},
			Operator: constraint.OpEqual,
			Value:    "opc_scan",
		},
	},
}

// TasksForSpecificRack restricts task retrieval to tasks belonging to a specific rack.
// Uses FieldConstraint to check the resource's rack field.
func TasksForSpecificRack(rackKey string) policy.Policy {
	return policy.Policy{
		Key:     uuid.New(),
		Name:    "tasks-for-rack-" + rackKey,
		Objects: []ontology.ID{{Type: "task"}},
		Actions: []access.Action{access.ActionRetrieve},
		Effect:  policy.EffectAllow,
		Constraints: []constraint.Constraint{
			constraint.Field{
				Target:   "resource",
				Field:    []string{"rack"},
				Operator: constraint.OpEqual,
				Value:    rackKey,
			},
		},
	}
}

// RangesWithLabels restricts range retrieval to ranges with specific labels.
// Uses RelationshipConstraint to check the "labeled_by" ontology relationship.
func RangesWithLabels(allowedLabels []ontology.ID) policy.Policy {
	return policy.Policy{
		Key:     uuid.New(),
		Name:    "ranges-with-labels",
		Objects: []ontology.ID{{Type: "range"}},
		Actions: []access.Action{access.ActionRetrieve},
		Effect:  policy.EffectAllow,
		Constraints: []constraint.Constraint{
			constraint.Relationship{
				Relationship: "labeled_by",
				Operator:     constraint.RelOpContainsSome,
				Value:        allowedLabels,
			},
		},
	}
}

// DenyAllWritesDuringTest blocks all write operations when system is in test mode.
// This is a DENY policy that takes precedence over allow policies.
// Uses FieldConstraint to check the system's mode field.
var DenyAllWritesDuringTest = policy.Policy{
	Key:     uuid.New(),
	Name:    "deny-writes-during-test",
	Objects: []ontology.ID{}, // Empty = all objects
	Actions: []access.Action{access.ActionCreate, access.ActionUpdate, access.ActionDelete},
	Effect:  policy.EffectDeny,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "system",
			Field:    []string{"mode"},
			Operator: constraint.OpEqual,
			Value:    "test",
		},
	},
}

// CombinedConstraints shows multiple constraints that must ALL be satisfied.
// User can only update draft schematics they created.
// Uses both RelationshipConstraint and FieldConstraint.
var CombinedConstraints = policy.Policy{
	Key:     uuid.New(),
	Name:    "update-own-draft-schematics",
	Objects: []ontology.ID{{Type: "schematic"}},
	Actions: []access.Action{access.ActionUpdate},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Relationship{
			Relationship: "created_by",
			Operator:     constraint.RelOpContainsSome,
			MatchSubject: true,
		},
		constraint.Field{
			Target:   "resource",
			Field:    []string{"status"},
			Operator: constraint.OpEqual,
			Value:    "draft",
		},
	},
}

// =============================================================================
// SECURITY SCENARIOS FROM TABLE
// =============================================================================

// -----------------------------------------------------------------------------
// Scenario: Channel data for Tuesday from 2-4 is available for a vendor to view
// Type: Security
// Status: SOLVED - uses FieldConstraint on request.time_range
// -----------------------------------------------------------------------------

// VendorTuesdayChannelAccess allows a vendor to read channel data only during
// a specific time window (Tuesday 2-4pm).
func VendorTuesdayChannelAccess(channelKey string, tuesdayWindow telem.TimeRange) policy.Policy {
	return policy.Policy{
		Key:     uuid.New(),
		Name:    "vendor-tuesday-channel-access",
		Objects: []ontology.ID{{Type: "channel", Key: channelKey}},
		Actions: []access.Action{access.ActionRetrieve},
		Effect:  policy.EffectAllow,
		Constraints: []constraint.Constraint{
			constraint.Field{
				Target:   "request",
				Field:    []string{"time_range"},
				Operator: constraint.OpWithin,
				Value:    tuesdayWindow,
			},
		},
	}
}

// -----------------------------------------------------------------------------
// Scenario: Engineers in LA can edit LA schematics, not Texas
// Type: Security (ABAC)
// Status: SOLVED - uses FieldConstraint on subject.location and resource.workspace
// -----------------------------------------------------------------------------

// LAEngineerSchematicAccess allows engineers at LA location to edit
// schematics in LA workspaces only.
var LAEngineerSchematicAccess = policy.Policy{
	Key:     uuid.New(),
	Name:    "la-engineer-schematic-access",
	Objects: []ontology.ID{{Type: "schematic"}},
	Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "subject",
			Field:    []string{"location"},
			Operator: constraint.OpEqual,
			Value:    "la",
		},
		constraint.Field{
			Target:   "resource",
			Field:    []string{"workspace"},
			Operator: constraint.OpEqual,
			Value:    "la",
		},
	},
}

// DenyTexasSchematicsForLA explicitly denies LA engineers from
// editing Texas schematics.
var DenyTexasSchematicsForLA = policy.Policy{
	Key:     uuid.New(),
	Name:    "deny-texas-schematics-for-la",
	Objects: []ontology.ID{{Type: "schematic"}},
	Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
	Effect:  policy.EffectDeny,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "subject",
			Field:    []string{"location"},
			Operator: constraint.OpEqual,
			Value:    "la",
		},
		constraint.Field{
			Target:   "resource",
			Field:    []string{"workspace"},
			Operator: constraint.OpEqual,
			Value:    "texas",
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Top secret clearance can read/write "top-secret", only read "secret"
// Type: Security (ABAC)
// Status: SOLVED - uses FieldConstraint on subject.clearance
// -----------------------------------------------------------------------------

// TopSecretFullAccess allows users with top-secret clearance to read/write
// top-secret classified data.
var TopSecretFullAccess = policy.Policy{
	Key:     uuid.New(),
	Name:    "top-secret-full-access",
	Objects: []ontology.ID{{Type: "channel"}},
	Actions: []access.Action{access.ActionRetrieve, access.ActionUpdate, access.ActionCreate},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "subject",
			Field:    []string{"clearance"},
			Operator: constraint.OpEqual,
			Value:    "top-secret",
		},
		constraint.Field{
			Target:   "resource",
			Field:    []string{"classification"},
			Operator: constraint.OpEqual,
			Value:    "top-secret",
		},
	},
}

// TopSecretReadSecret allows users with top-secret clearance to only READ
// secret classified data (not write).
var TopSecretReadSecret = policy.Policy{
	Key:     uuid.New(),
	Name:    "top-secret-read-secret",
	Objects: []ontology.ID{{Type: "channel"}},
	Actions: []access.Action{access.ActionRetrieve},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "subject",
			Field:    []string{"clearance"},
			Operator: constraint.OpEqual,
			Value:    "top-secret",
		},
		constraint.Field{
			Target:   "resource",
			Field:    []string{"classification"},
			Operator: constraint.OpEqual,
			Value:    "secret",
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Secret clearance can read/write "secret", only read "top-secret"
// Type: Security (ABAC)
// Status: SOLVED - uses FieldConstraint on subject.clearance
// -----------------------------------------------------------------------------

// SecretFullAccess allows users with secret clearance to read/write
// secret classified data.
var SecretFullAccess = policy.Policy{
	Key:     uuid.New(),
	Name:    "secret-full-access",
	Objects: []ontology.ID{{Type: "channel"}},
	Actions: []access.Action{access.ActionRetrieve, access.ActionUpdate, access.ActionCreate},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "subject",
			Field:    []string{"clearance"},
			Operator: constraint.OpEqual,
			Value:    "secret",
		},
		constraint.Field{
			Target:   "resource",
			Field:    []string{"classification"},
			Operator: constraint.OpEqual,
			Value:    "secret",
		},
	},
}

// DenySecretWriteTopSecret denies users with secret clearance from writing
// top-secret data.
var DenySecretWriteTopSecret = policy.Policy{
	Key:     uuid.New(),
	Name:    "deny-secret-write-top-secret",
	Objects: []ontology.ID{{Type: "channel"}},
	Actions: []access.Action{access.ActionUpdate, access.ActionCreate},
	Effect:  policy.EffectDeny,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "subject",
			Field:    []string{"clearance"},
			Operator: constraint.OpEqual,
			Value:    "secret",
		},
		constraint.Field{
			Target:   "resource",
			Field:    []string{"classification"},
			Operator: constraint.OpEqual,
			Value:    "top-secret",
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Shutting down iterators while a test is going on
// Type: Security
// Status: SOLVED - uses FieldConstraint on system.mode
// -----------------------------------------------------------------------------

// DenyIteratorsDuringTest blocks all iterator/read operations during test mode.
var DenyIteratorsDuringTest = policy.Policy{
	Key:     uuid.New(),
	Name:    "deny-iterators-during-test",
	Objects: []ontology.ID{{Type: "channel"}},
	Actions: []access.Action{access.ActionRetrieve},
	Effect:  policy.EffectDeny,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "system",
			Field:    []string{"mode"},
			Operator: constraint.OpEqual,
			Value:    "test",
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Block accessing data on the holiday break
// Type: Security
// Status: SOLVED - uses FieldConstraint on system.current_time
// -----------------------------------------------------------------------------

// DenyAccessDuringHoliday blocks all data access during a holiday period.
func DenyAccessDuringHoliday(holidayPeriod telem.TimeRange) policy.Policy {
	return policy.Policy{
		Key:     uuid.New(),
		Name:    "deny-access-during-holiday",
		Objects: []ontology.ID{},
		Actions: []access.Action{access.ActionRetrieve, access.ActionUpdate, access.ActionCreate, access.ActionDelete},
		Effect:  policy.EffectDeny,
		Constraints: []constraint.Constraint{
			constraint.Field{
				Target:   "system",
				Field:    []string{"current_time"},
				Operator: constraint.OpWithin,
				Value:    holidayPeriod,
			},
		},
	}
}

// -----------------------------------------------------------------------------
// Scenario: Certain ontology relationships (default groups) cannot be edited
// Type: Security
// Status: SOLVED - uses FieldConstraint on resource.internal
// -----------------------------------------------------------------------------

// DenyEditBuiltinGroups prevents editing of built-in/internal groups.
var DenyEditBuiltinGroups = policy.Policy{
	Key:     uuid.New(),
	Name:    "deny-edit-builtin-groups",
	Objects: []ontology.ID{{Type: "group"}},
	Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
	Effect:  policy.EffectDeny,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "resource",
			Field:    []string{"internal"},
			Operator: constraint.OpEqual,
			Value:    true,
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Ontology relationships between tasks and statuses cannot be edited
// Type: Security
// Status: SOLVED - uses FieldConstraint on resource.owner_type
// -----------------------------------------------------------------------------

// DenyEditTaskStatusRelationships prevents editing relationships owned by tasks.
var DenyEditTaskStatusRelationships = policy.Policy{
	Key:     uuid.New(),
	Name:    "deny-edit-task-status-relationships",
	Objects: []ontology.ID{{Type: "relationship"}},
	Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
	Effect:  policy.EffectDeny,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "resource",
			Field:    []string{"owner_type"},
			Operator: constraint.OpIn,
			Value:    []string{"task", "arc"},
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Embedded rack can only be edited via backend
// Type: Security
// Status: SOLVED - uses FieldConstraint on request.source
// -----------------------------------------------------------------------------

// DenyEmbeddedRackEditFromClients prevents editing embedded rack from client apps.
var DenyEmbeddedRackEditFromClients = policy.Policy{
	Key:     uuid.New(),
	Name:    "deny-embedded-rack-edit-from-clients",
	Objects: []ontology.ID{{Type: "rack"}},
	Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
	Effect:  policy.EffectDeny,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "resource",
			Field:    []string{"owner"},
			Operator: constraint.OpEqual,
			Value:    "node1",
		},
		constraint.Field{
			Target:   "request",
			Field:    []string{"source"},
			Operator: constraint.OpIn,
			Value:    []string{"console", "pluto"},
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: User can't rename sy_node_1_metrics channels
// Type: Security
// Status: SOLVED - uses FieldConstraint on resource.owner and request.properties
// -----------------------------------------------------------------------------

// DenyRenameNodeChannels prevents renaming channels owned by nodes.
var DenyRenameNodeChannels = policy.Policy{
	Key:     uuid.New(),
	Name:    "deny-rename-node-channels",
	Objects: []ontology.ID{{Type: "channel"}},
	Actions: []access.Action{access.ActionUpdate},
	Effect:  policy.EffectDeny,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "resource",
			Field:    []string{"owner"},
			Operator: constraint.OpEqual,
			Value:    "node1",
		},
		constraint.Field{
			Target:   "request",
			Field:    []string{"properties"},
			Operator: constraint.OpContains,
			Value:    "name",
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: User wants to "lock" a workspace to only be edited by them
// Type: Security (ReBAC)
// Status: SOLVED - uses RelationshipConstraint for created_by
// -----------------------------------------------------------------------------

// LockedWorkspaceCreatorOnly allows only the creator to edit a locked workspace.
var LockedWorkspaceCreatorOnly = policy.Policy{
	Key:     uuid.New(),
	Name:    "locked-workspace-creator-only",
	Objects: []ontology.ID{{Type: "workspace"}},
	Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "resource",
			Field:    []string{"locked"},
			Operator: constraint.OpEqual,
			Value:    true,
		},
		constraint.Relationship{
			Relationship: "created_by",
			Operator:     constraint.RelOpContainsSome,
			MatchSubject: true,
		},
	},
}

// =============================================================================
// UX SCENARIOS FROM TABLE (for filtering, not hard security)
// =============================================================================

// -----------------------------------------------------------------------------
// Scenario: User can retrieve sy_task_set via Pluto, but not via Console
// Type: UX
// Status: SOLVED - uses FieldConstraint on request.source
// -----------------------------------------------------------------------------

// AllowTaskSetFromPlutoOnly allows retrieving task set only from Pluto client.
var AllowTaskSetFromPlutoOnly = policy.Policy{
	Key:     uuid.New(),
	Name:    "allow-task-set-from-pluto-only",
	Objects: []ontology.ID{{Type: "channel", Key: "sy_task_set"}},
	Actions: []access.Action{access.ActionRetrieve},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "request",
			Field:    []string{"source"},
			Operator: constraint.OpEqual,
			Value:    "pluto",
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Task statuses can be updated via Driver, not Console
// Type: UX
// Status: SOLVED - uses FieldConstraint on request.source
// -----------------------------------------------------------------------------

// AllowTaskStatusUpdateFromDriver allows task status updates only from Driver.
var AllowTaskStatusUpdateFromDriver = policy.Policy{
	Key:     uuid.New(),
	Name:    "allow-task-status-update-from-driver",
	Objects: []ontology.ID{{Type: "status"}},
	Actions: []access.Action{access.ActionUpdate},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "resource",
			Field:    []string{"owner_type"},
			Operator: constraint.OpEqual,
			Value:    "task",
		},
		constraint.Field{
			Target:   "request",
			Field:    []string{"source"},
			Operator: constraint.OpEqual,
			Value:    "driver",
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: User can see sy_node_1_metrics but not sy_task_cmd in Console
// Type: UX
// Status: SOLVED - uses FieldConstraint on resource.owner_type
// -----------------------------------------------------------------------------

// AllowUserVisibleChannels allows viewing channels not owned by builtin/system.
var AllowUserVisibleChannels = policy.Policy{
	Key:     uuid.New(),
	Name:    "allow-user-visible-channels",
	Objects: []ontology.ID{{Type: "channel"}},
	Actions: []access.Action{access.ActionRetrieve},
	Effect:  policy.EffectAllow,
	Constraints: []constraint.Constraint{
		constraint.Field{
			Target:   "resource",
			Field:    []string{"owner_type"},
			Operator: constraint.OpNotIn,
			Value:    []string{"builtin", "system"},
		},
	},
}

// =============================================================================
// COMPUTED CONSTRAINT EXAMPLES
// =============================================================================

// MaxDurationDataRead limits data reads to a maximum duration.
// Uses ComputedConstraint to check the duration of the time range.
func MaxDurationDataRead(maxDuration telem.TimeSpan) policy.Policy {
	return policy.Policy{
		Key:     uuid.New(),
		Name:    "max-duration-data-read",
		Objects: []ontology.ID{{Type: "channel"}},
		Actions: []access.Action{access.ActionRetrieve},
		Effect:  policy.EffectAllow,
		Constraints: []constraint.Constraint{
			constraint.Computed{
				Property: "duration",
				Source:   []string{"request", "time_range"},
				Operator: constraint.OpLessThanOrEqual,
				Value:    maxDuration,
			},
		},
	}
}

// =============================================================================
// SUMMARY OF CONSTRAINT TYPES
// =============================================================================
//
// FieldConstraint:
//   Target: "resource" | "subject" | "request" | "system"
//   Field:  []string path to the field (e.g., ["status"], ["clearance"])
//   Operators: OpEqual, OpNotEqual, OpIn, OpNotIn, OpContains, OpContainsAny,
//              OpWithin, OpSubsetOf, OpLessThan, OpLessThanOrEqual,
//              OpGreaterThan, OpGreaterThanOrEqual
//
// RelationshipConstraint:
//   Relationship: "created_by" | "labeled_by" | "parent_of" | "member_of" | etc.
//   Operators: RelOpContainsSome, RelOpContainsAll, RelOpContainsNone
//   Value: []ontology.ID - the IDs to match against
//   MatchSubject: bool - when true, matches against the requesting subject instead of Value
//
// ComputedConstraint:
//   Property: "duration" | "age" | "count"
//   Source:   []string path to source value
//   Operators: comparison operators (lt, lte, gt, gte, eq)
//
// =============================================================================

// =============================================================================
// ENFORCE vs FILTER USAGE
// =============================================================================
//
// The Enforcer interface provides two methods for different use cases:
//
// 1. Enforce(ctx, req) error
//    - Returns ErrDenied if ANY object in the request is not accessible
//    - Use for operations that should fail completely if access is denied
//    - Examples: update, delete, create operations
//
//    err := enforcer.Enforce(ctx, access.Request{
//        Subject: userID,
//        Objects: []ontology.ID{schematicID},
//        Action:  access.ActionDelete,
//    })
//    if errors.Is(err, access.ErrDenied) {
//        return errors.New("you don't have permission to delete this schematic")
//    }
//
// 2. Filter(ctx, req) ([]ontology.ID, error)
//    - Returns only the objects the subject has access to
//    - Does NOT fail on denied objects - simply excludes them
//    - Use for search/list operations where partial results are acceptable
//
//    // User searches for channels - some may be restricted
//    allChannels := searchChannels(query)
//    accessibleChannels, err := enforcer.Filter(ctx, access.Request{
//        Subject: userID,
//        Objects: allChannels,
//        Action:  access.ActionRetrieve,
//    })
//    // accessibleChannels contains only channels the user can see
//
// =============================================================================
// FILTERING EXAMPLE: Channel Search
// =============================================================================
//
// Scenario: User searches for channels. The search should not fail if they
// don't have access to some channels - it should just filter them out.
//
// Policy: Users can only see channels in their workspace
//
//   policy.Policy{
//       Name:    "workspace-channel-access",
//       Objects: []ontology.ID{{Type: "channel"}},
//       Actions: []access.Action{access.ActionRetrieve},
//       Effect:  policy.EffectAllow,
//       Constraints: []constraint.Constraint{
//           constraint.Field{
//               Target:   "resource",
//               Field:    []string{"workspace"},
//               Operator: constraint.OpEqual,
//               Value:    "user-workspace",  // Would be dynamic per-user
//           },
//       },
//   }
//
// Usage in channel service:
//
//   func (s *Service) Search(ctx context.Context, userID ontology.ID, query string) ([]Channel, error) {
//       // 1. Get all channels matching the query (no access filtering yet)
//       allChannels, err := s.db.SearchChannels(query)
//       if err != nil {
//           return nil, err
//       }
//
//       // 2. Extract channel IDs
//       channelIDs := make([]ontology.ID, len(allChannels))
//       for i, ch := range allChannels {
//           channelIDs[i] = ch.OntologyID()
//       }
//
//       // 3. Filter to only accessible channels
//       accessibleIDs, err := s.enforcer.Filter(ctx, access.Request{
//           Subject: userID,
//           Objects: channelIDs,
//           Action:  access.ActionRetrieve,
//       })
//       if err != nil {
//           return nil, err
//       }
//
//       // 4. Return only accessible channels
//       accessibleSet := make(map[string]bool)
//       for _, id := range accessibleIDs {
//           accessibleSet[id.Key] = true
//       }
//       var result []Channel
//       for _, ch := range allChannels {
//           if accessibleSet[ch.Key()] {
//               result = append(result, ch)
//           }
//       }
//       return result, nil
//   }
//
// =============================================================================
// FILTERING EXAMPLE: Schematic List with Creator-Only Edit
// =============================================================================
//
// Scenario: All users can view all schematics, but can only edit their own.
// When listing schematics, we want to indicate which ones the user can edit.
//
// Policies:
//
//   // Everyone can view schematics
//   policy.Policy{
//       Name:    "view-all-schematics",
//       Objects: []ontology.ID{{Type: "schematic"}},
//       Actions: []access.Action{access.ActionRetrieve},
//       Effect:  policy.EffectAllow,
//   }
//
//   // Only creator can edit
//   policy.Policy{
//       Name:    "edit-own-schematics",
//       Objects: []ontology.ID{{Type: "schematic"}},
//       Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
//       Effect:  policy.EffectAllow,
//       Constraints: []constraint.Constraint{
//           constraint.Relationship{
//               Relationship: "created_by",
//               Operator:     constraint.RelOpContainsSome,
//               MatchSubject: true,
//           },
//       },
//   }
//
// Usage in schematic service:
//
//   func (s *Service) ListWithPermissions(ctx context.Context, userID ontology.ID) ([]SchematicWithPerms, error) {
//       schematics, _ := s.db.ListAll()
//       schematicIDs := extractIDs(schematics)
//
//       // Check which ones user can edit
//       editableIDs, _ := s.enforcer.Filter(ctx, access.Request{
//           Subject: userID,
//           Objects: schematicIDs,
//           Action:  access.ActionUpdate,
//       })
//       editableSet := toSet(editableIDs)
//
//       // Return schematics with edit permission flag
//       result := make([]SchematicWithPerms, len(schematics))
//       for i, sch := range schematics {
//           result[i] = SchematicWithPerms{
//               Schematic: sch,
//               CanEdit:   editableSet[sch.Key()],
//           }
//       }
//       return result, nil
//   }
//
// =============================================================================
