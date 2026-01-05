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
	Constraints: []policy.Constraint{
		policy.RelationshipConstraint{
			Relationship: "created_by",
			Operator:     policy.OpEqualSubject,
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
		Constraints: []policy.Constraint{
			policy.FieldConstraint{
				Target:   "request",
				Field:    []string{"time_range"},
				Operator: policy.OpWithin,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "request",
			Field:    []string{"properties"},
			Operator: policy.OpSubsetOf,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"status"},
			Operator: policy.OpEqual,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"type"},
			Operator: policy.OpEqual,
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
		Constraints: []policy.Constraint{
			policy.FieldConstraint{
				Target:   "resource",
				Field:    []string{"rack"},
				Operator: policy.OpEqual,
				Value:    rackKey,
			},
		},
	}
}

// RangesWithLabels restricts range retrieval to ranges with specific labels.
// Uses RelationshipConstraint to check the "labeled_by" ontology relationship.
func RangesWithLabels(allowedLabels []string) policy.Policy {
	return policy.Policy{
		Key:     uuid.New(),
		Name:    "ranges-with-labels",
		Objects: []ontology.ID{{Type: "range"}},
		Actions: []access.Action{access.ActionRetrieve},
		Effect:  policy.EffectAllow,
		Constraints: []policy.Constraint{
			policy.RelationshipConstraint{
				Relationship: "labeled_by",
				Operator:     policy.OpContainsAny,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "system",
			Field:    []string{"mode"},
			Operator: policy.OpEqual,
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
	Constraints: []policy.Constraint{
		policy.RelationshipConstraint{
			Relationship: "created_by",
			Operator:     policy.OpEqualSubject,
		},
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"status"},
			Operator: policy.OpEqual,
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
		Constraints: []policy.Constraint{
			policy.FieldConstraint{
				Target:   "request",
				Field:    []string{"time_range"},
				Operator: policy.OpWithin,
				Value:    tuesdayWindow,
			},
		},
	}
}

// -----------------------------------------------------------------------------
// Scenario: Engineers in Kinloss can edit Kinloss schematics, not Faxie
// Type: Security (ABAC)
// Status: SOLVED - uses FieldConstraint on subject.location and resource.workspace
// -----------------------------------------------------------------------------

// KinlossEngineerSchematicAccess allows engineers at Kinloss location to edit
// schematics in Kinloss workspaces only.
var KinlossEngineerSchematicAccess = policy.Policy{
	Key:     uuid.New(),
	Name:    "kinloss-engineer-schematic-access",
	Objects: []ontology.ID{{Type: "schematic"}},
	Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
	Effect:  policy.EffectAllow,
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "subject",
			Field:    []string{"location"},
			Operator: policy.OpEqual,
			Value:    "kinloss",
		},
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"workspace"},
			Operator: policy.OpEqual,
			Value:    "kinloss",
		},
	},
}

// DenyFaxieSchematicsForKinloss explicitly denies Kinloss engineers from
// editing Faxie schematics.
var DenyFaxieSchematicsForKinloss = policy.Policy{
	Key:     uuid.New(),
	Name:    "deny-faxie-schematics-for-kinloss",
	Objects: []ontology.ID{{Type: "schematic"}},
	Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
	Effect:  policy.EffectDeny,
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "subject",
			Field:    []string{"location"},
			Operator: policy.OpEqual,
			Value:    "kinloss",
		},
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"workspace"},
			Operator: policy.OpEqual,
			Value:    "faxie",
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "subject",
			Field:    []string{"clearance"},
			Operator: policy.OpEqual,
			Value:    "top-secret",
		},
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"classification"},
			Operator: policy.OpEqual,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "subject",
			Field:    []string{"clearance"},
			Operator: policy.OpEqual,
			Value:    "top-secret",
		},
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"classification"},
			Operator: policy.OpEqual,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "subject",
			Field:    []string{"clearance"},
			Operator: policy.OpEqual,
			Value:    "secret",
		},
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"classification"},
			Operator: policy.OpEqual,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "subject",
			Field:    []string{"clearance"},
			Operator: policy.OpEqual,
			Value:    "secret",
		},
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"classification"},
			Operator: policy.OpEqual,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "system",
			Field:    []string{"mode"},
			Operator: policy.OpEqual,
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
		Constraints: []policy.Constraint{
			policy.FieldConstraint{
				Target:   "system",
				Field:    []string{"current_time"},
				Operator: policy.OpWithin,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"internal"},
			Operator: policy.OpEqual,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"owner_type"},
			Operator: policy.OpIn,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"owner"},
			Operator: policy.OpEqual,
			Value:    "node1",
		},
		policy.FieldConstraint{
			Target:   "request",
			Field:    []string{"source"},
			Operator: policy.OpIn,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"owner"},
			Operator: policy.OpEqual,
			Value:    "node1",
		},
		policy.FieldConstraint{
			Target:   "request",
			Field:    []string{"properties"},
			Operator: policy.OpContains,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"locked"},
			Operator: policy.OpEqual,
			Value:    true,
		},
		policy.RelationshipConstraint{
			Relationship: "created_by",
			Operator:     policy.OpEqualSubject,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "request",
			Field:    []string{"source"},
			Operator: policy.OpEqual,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"owner_type"},
			Operator: policy.OpEqual,
			Value:    "task",
		},
		policy.FieldConstraint{
			Target:   "request",
			Field:    []string{"source"},
			Operator: policy.OpEqual,
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
	Constraints: []policy.Constraint{
		policy.FieldConstraint{
			Target:   "resource",
			Field:    []string{"owner_type"},
			Operator: policy.OpNotIn,
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
		Constraints: []policy.Constraint{
			policy.ComputedConstraint{
				Property: "duration",
				Source:   []string{"request", "time_range"},
				Operator: policy.OpLessThanOrEqual,
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
//   Operators: OpEqualSubject, OpEqual, OpIn, OpContainsAny
//
// ComputedConstraint:
//   Property: "duration" | "age" | "count"
//   Source:   []string path to source value
//   Operators: comparison operators (lt, lte, gt, gte, eq)
//
// =============================================================================
