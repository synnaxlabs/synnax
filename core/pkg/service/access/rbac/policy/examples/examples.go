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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic"
	"github.com/synnaxlabs/x/telem"
)

// =============================================================================
// CONSTRAINT KINDS:
//
// 1. KindField - Direct field access on resources, subjects, requests, system
//    Example: resource.status == "active", subject.clearance in ["secret", "top_secret"]
//
// 2. KindRelationship - Ontology graph traversal
//    Example: resource created_by subject, resource labeled_by "production"
//
// 3. KindComputed - Derived/calculated values
//    Example: request.time_range duration <= 24h
//
// 4. KindAnd - All child constraints must be satisfied
// 5. KindOr - At least one child constraint must be satisfied
// 6. KindNot - Inverts the result of a child constraint
// =============================================================================

// PolicyOnlyUpdateSchematicsCreatedByUsers allows users to only update schematics they
// created.
var PolicyOnlyUpdateSchematicsCreatedByUsers = policy.Policy{
	Key:    uuid.New(),
	Name:   "Schematics can only be updated by their creator",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:    constraint.KindRelationship,
		Objects: []ontology.ID{{Type: schematic.OntologyType}},
		Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
		Relationship: ontology.CreatedBy,
		Operator:     constraint.OpContainsAny,
		MatchSubject: true,
	},
}

// PolicyTimeRangeChannelRead restricts channel data access to a specific time range.
func PolicyTimeRangeChannelRead(channelKey channel.Key, allowedRange telem.TimeRange) policy.Policy {
	return policy.Policy{
		Key:    uuid.New(),
		Name:   "Channel data can only be read within a specific time range",
		Effect: policy.EffectAllow,
		Constraint: constraint.Constraint{
			Kind:     constraint.KindField,
			Objects:  []ontology.ID{channel.OntologyID(channelKey)},
			Actions:  []access.Action{"read_data"},
			Target:   "request",
			Field:    []string{"time_range"},
			Operator: constraint.OpWithin,
			Value:    allowedRange,
		},
	}
}

// PolicyDenyBuiltinStatusNameKeyEdit prevents editing Name or Key on builtin-owned
// statuses.
var PolicyDenyBuiltinStatusNameKeyEdit = policy.Policy{
	Key:    uuid.New(),
	Name:   "Cannot edit Name or Key on builtin-owned statuses",
	Effect: policy.EffectDeny,
	Constraint: constraint.Constraint{
		Kind:    constraint.KindAnd,
		Objects: []ontology.ID{{Type: status.OntologyType}},
		Actions: []access.Action{access.ActionUpdate},
		Constraints: []constraint.Constraint{
			{
				Kind:            constraint.KindRelationship,
				Relationship:    ontology.CreatedBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: ontology.TypeBuiltIn}},
			},
			{
				Kind:     constraint.KindField,
				Target:   "request",
				Field:    []string{"properties"},
				Operator: constraint.OpContainsAny,
				Value:    []string{"Name", "Key"},
			},
		},
	},
}

// ErrorStatusesOnly restricts retrieval to statuses in error state.
var ErrorStatusesOnly = policy.Policy{
	Key:    uuid.New(),
	Name:   "error-statuses-only",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:     constraint.KindField,
		Objects:  []ontology.ID{{Type: "status"}},
		Actions:  []access.Action{access.ActionRetrieve},
		Target:   "resource",
		Field:    []string{"status"},
		Operator: constraint.OpEqual,
		Value:    "error",
	},
}

// OPCScanTasksOnly restricts task retrieval to opc_scan type tasks. Uses
// KindField to check the resource's type field.
var OPCScanTasksOnly = policy.Policy{
	Key:    uuid.New(),
	Name:   "opc-scan-tasks-only",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:     constraint.KindField,
		Objects:  []ontology.ID{{Type: "task"}},
		Actions:  []access.Action{access.ActionRetrieve},
		Target:   "resource",
		Field:    []string{"type"},
		Operator: constraint.OpEqual,
		Value:    "opc_scan",
	},
}

// TasksForSpecificRack restricts task retrieval to tasks belonging to a specific rack.
// Uses KindComputed to check the resource's rack field.
func TasksForSpecificRack(rackKey string) policy.Policy {
	return policy.Policy{
		Key:    uuid.New(),
		Name:   "tasks-for-rack-" + rackKey,
		Effect: policy.EffectAllow,
		Constraint: constraint.Constraint{
			Kind:     constraint.KindComputed,
			Objects:  []ontology.ID{{Type: task.OntologyType}},
			Actions:  []access.Action{access.ActionRetrieve},
			Property: "rack",
			Source:   []string{"resource", "key"},
			Operator: constraint.OpEqual,
			Value:    rackKey,
		},
	}
}

// RangesWithLabels restricts range retrieval to ranges with specific labels. Uses
// KindRelationship to check the "labeled_by" ontology relationship.
func RangesWithLabels(allowedLabels []ontology.ID) policy.Policy {
	return policy.Policy{
		Key:    uuid.New(),
		Name:   "ranges-with-labels",
		Effect: policy.EffectAllow,
		Constraint: constraint.Constraint{
			Kind:            constraint.KindRelationship,
			Objects:         []ontology.ID{{Type: ranger.OntologyType}},
			Actions:         []access.Action{access.ActionRetrieve},
			Relationship:    label.LabeledBy,
			Operator:        constraint.OpContainsAny,
			RelationshipIDs: allowedLabels,
		},
	}
}

// DenyAllWritesDuringTest blocks all write operations when system is in test mode. This
// is a DENY policy that takes precedence over allow policies. Uses KindField to
// check the system's mode field.
func DenyAllWritesDuringTest(testTime telem.TimeRange) policy.Policy {
	return policy.Policy{
		Key:    uuid.New(),
		Name:   "deny-writes-during-test",
		Effect: policy.EffectDeny,
		Constraint: constraint.Constraint{
			Kind:     constraint.KindField,
			Objects:  []ontology.ID{}, // Empty = all objects
			Actions:  []access.Action{access.ActionCreate, access.ActionUpdate, access.ActionDelete},
			Target:   "system",
			Field:    []string{"time"},
			Operator: constraint.OpWithin,
			Value:    testTime,
		},
	}
}

// =============================================================================
// SECURITY SCENARIOS FROM TABLE
// =============================================================================

// -----------------------------------------------------------------------------
// Scenario: Channel data for Tuesday from 2-4 is available for a vendor to view
// Type: Security
// Status: SOLVED - uses KindField on request.time_range
// -----------------------------------------------------------------------------

// VendorTuesdayChannelAccess allows a vendor to read channel data only during
// a specific time window (Tuesday 2-4pm).
func VendorTuesdayChannelAccess(channelKey string, tuesdayWindow telem.TimeRange) policy.Policy {
	return policy.Policy{
		Key:    uuid.New(),
		Name:   "vendor-tuesday-channel-access",
		Effect: policy.EffectAllow,
		Constraint: constraint.Constraint{
			Kind:     constraint.KindField,
			Objects:  []ontology.ID{{Type: "channel", Key: channelKey}},
			Actions:  []access.Action{access.ActionRetrieve},
			Target:   "request",
			Field:    []string{"time_range"},
			Operator: constraint.OpWithin,
			Value:    tuesdayWindow,
		},
	}
}

// -----------------------------------------------------------------------------
// Scenario: Engineers in LA can edit LA schematics, not Texas
// Type: Security (ABAC)
// Status: SOLVED - uses KindRelationship on subject and resource labels
// -----------------------------------------------------------------------------

// LAEngineerSchematicAccess allows engineers at LA location to edit
// schematics in LA workspaces only.
var LAEngineerSchematicAccess = policy.Policy{
	Key:    uuid.New(),
	Name:   "la-engineer-schematic-access",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:    constraint.KindAnd,
		Objects: []ontology.ID{{Type: "schematic"}},
		Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
		Constraints: []constraint.Constraint{
			{
				Kind:            constraint.KindRelationship,
				Target:          "subject",
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "la"}},
			},
			{
				Kind:         constraint.KindRelationship,
				Relationship: ontology.CreatedBy,
				Operator:     constraint.OpContainsAny,
				MatchSubject: true,
			},
		},
	},
}

// DenyTexasSchematicsForLA explicitly denies LA engineers from
// editing Texas schematics.
var DenyTexasSchematicsForLA = policy.Policy{
	Key:    uuid.New(),
	Name:   "deny-texas-schematics-for-la",
	Effect: policy.EffectDeny,
	Constraint: constraint.Constraint{
		Kind:    constraint.KindAnd,
		Objects: []ontology.ID{{Type: "schematic"}},
		Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
		Constraints: []constraint.Constraint{
			{
				Kind:            constraint.KindRelationship,
				Target:          "subject",
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "la"}},
			},
			{
				Kind:            constraint.KindRelationship,
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsNone,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "texas"}},
			},
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Top secret clearance can read/write "top-secret", only read "secret"
// Type: Security (ABAC)
// Status: REVISED - uses labels on subject and resource instead of field constraints
// -----------------------------------------------------------------------------

// TopSecretFullAccess allows subjects labeled "top-secret" (clearance) to read/write
// objects labeled "top-secret" (classification).
var TopSecretFullAccess = policy.Policy{
	Key:    uuid.New(),
	Name:   "top-secret-full-access",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:    constraint.KindAnd,
		Objects: []ontology.ID{{Type: "channel"}},
		Actions: []access.Action{access.ActionRetrieve, access.ActionUpdate, access.ActionCreate},
		Constraints: []constraint.Constraint{
			// Subject must be labeled as "top-secret"
			{
				Kind:            constraint.KindRelationship,
				Target:          "subject",
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "top-secret"}},
			},
			// Resource (object) must be labeled as "top-secret"
			{
				Kind:            constraint.KindRelationship,
				Target:          "resource",
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "top-secret"}},
			},
		},
	},
}

// TopSecretReadSecret allows users with top-secret clearance to only READ
// secret classified data (not write).
var TopSecretReadSecret = policy.Policy{
	Key:    uuid.New(),
	Name:   "top-secret-read-secret",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:    constraint.KindAnd,
		Objects: []ontology.ID{{Type: "channel"}},
		Actions: []access.Action{access.ActionRetrieve},
		Constraints: []constraint.Constraint{
			{
				Kind:            constraint.KindRelationship,
				Target:          "subject",
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "top-secret"}},
			},
			{
				Kind:            constraint.KindRelationship,
				Target:          "resource",
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "secret"}},
			},
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Secret clearance can read/write "secret", only read "top-secret"
// Type: Security (ABAC)
// Status: SOLVED - uses labels on subject and resource
// -----------------------------------------------------------------------------

// SecretFullAccess allows users with secret clearance to read/write
// secret classified data.
var SecretFullAccess = policy.Policy{
	Key:    uuid.New(),
	Name:   "secret-full-access",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:    constraint.KindAnd,
		Objects: []ontology.ID{{Type: "channel"}},
		Actions: []access.Action{access.ActionRetrieve, access.ActionUpdate, access.ActionCreate},
		Constraints: []constraint.Constraint{
			{
				Kind:            constraint.KindRelationship,
				Target:          "subject",
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "secret"}},
			},
			{
				Kind:            constraint.KindRelationship,
				Target:          "resource",
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "secret"}},
			},
		},
	},
}

// DenySecretWriteTopSecret denies users with secret clearance from writing
// top-secret data.
var DenySecretWriteTopSecret = policy.Policy{
	Key:    uuid.New(),
	Name:   "deny-secret-write-top-secret",
	Effect: policy.EffectDeny,
	Constraint: constraint.Constraint{
		Kind:    constraint.KindAnd,
		Objects: []ontology.ID{{Type: "channel"}},
		Actions: []access.Action{access.ActionUpdate, access.ActionCreate},
		Constraints: []constraint.Constraint{
			{
				Kind:            constraint.KindRelationship,
				Target:          "subject",
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "secret"}},
			},
			{
				Kind:            constraint.KindRelationship,
				Target:          "resource",
				Relationship:    label.LabeledBy,
				Operator:        constraint.OpContainsAny,
				RelationshipIDs: []ontology.ID{{Type: label.OntologyType, Key: "top-secret"}},
			},
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Shutting down iterators while a test is going on
// Type: Security
// Status: SOLVED - uses KindField on system.mode
// -----------------------------------------------------------------------------

// DenyIteratorsDuringTest blocks all iterator/read operations during test mode.
var DenyIteratorsDuringTest = policy.Policy{
	Key:    uuid.New(),
	Name:   "deny-iterators-during-test",
	Effect: policy.EffectDeny,
	Constraint: constraint.Constraint{
		Kind:     constraint.KindField,
		Objects:  []ontology.ID{{Type: "channel"}},
		Actions:  []access.Action{"iterator"},
		Target:   "system",
		Field:    []string{"mode"},
		Operator: constraint.OpEqual,
		Value:    "test",
	},
}

// -----------------------------------------------------------------------------
// Scenario: Block accessing data on the holiday break
// Type: Security
// Status: SOLVED - uses KindField on system.current_time
// -----------------------------------------------------------------------------

// DenyAccessDuringHoliday blocks all data access during a holiday period.
func DenyAccessDuringHoliday(holidayPeriod telem.TimeRange) policy.Policy {
	return policy.Policy{
		Key:    uuid.New(),
		Name:   "deny-access-during-holiday",
		Effect: policy.EffectDeny,
		Constraint: constraint.Constraint{
			Kind:     constraint.KindField,
			Objects:  []ontology.ID{},
			Actions:  []access.Action{access.ActionRetrieve, access.ActionUpdate, access.ActionCreate, access.ActionDelete},
			Target:   "system",
			Field:    []string{"time"},
			Operator: constraint.OpWithin,
			Value:    holidayPeriod,
		},
	}
}

// -----------------------------------------------------------------------------
// Scenario: Certain ontology relationships (default groups) cannot be edited
// Type: Security
// Status: SOLVED - uses KindRelationship on resource.internal
// -----------------------------------------------------------------------------

// DenyEditBuiltinGroups prevents editing of built-in/internal groups.
var DenyEditBuiltinGroups = policy.Policy{
	Key:    uuid.New(),
	Name:   "deny-edit-builtin-groups",
	Effect: policy.EffectDeny,
	Constraint: constraint.Constraint{
		Kind:            constraint.KindRelationship,
		Objects:         []ontology.ID{{Type: "group"}},
		Actions:         []access.Action{access.ActionUpdate, access.ActionDelete},
		Relationship:    ontology.CreatedBy,
		Operator:        constraint.OpContainsNone,
		RelationshipIDs: []ontology.ID{{Type: ontology.TypeBuiltIn}},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Ontology relationships between tasks and statuses cannot be edited
// Type: Security
// Status: SOLVED - uses KindRelationship on resource.owner_type
// -----------------------------------------------------------------------------

// DenyEditTaskStatusRelationships prevents editing relationships owned by tasks.
var DenyEditTaskStatusRelationships = policy.Policy{
	Key:    uuid.New(),
	Name:   "deny-edit-task-status-relationships",
	Effect: policy.EffectDeny,
	Constraint: constraint.Constraint{
		Kind:            constraint.KindRelationship,
		Objects:         []ontology.ID{{Type: "relationship"}},
		Actions:         []access.Action{access.ActionUpdate, access.ActionDelete},
		Relationship:    ontology.CreatedBy,
		Operator:        constraint.OpContainsAny,
		RelationshipIDs: []ontology.ID{{Type: task.OntologyType}, {Type: arc.OntologyType}},
	},
}

// -----------------------------------------------------------------------------
// Scenario: Embedded rack can only be edited via backend
// Type: Security
// Status: SOLVED - uses KindField on request.source
// -----------------------------------------------------------------------------

// DenyEmbeddedRackEditFromClients prevents editing embedded rack from client apps.
var DenyEmbeddedRackEditFromClients = policy.Policy{
	Key:    uuid.New(),
	Name:   "deny-embedded-rack-edit-from-clients",
	Effect: policy.EffectDeny,
	Constraint: constraint.Constraint{
		Kind:    constraint.KindAnd,
		Objects: []ontology.ID{{Type: "rack"}},
		Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
		Constraints: []constraint.Constraint{
			{
				Kind:     constraint.KindField,
				Target:   "resource",
				Field:    []string{"owner"},
				Operator: constraint.OpEqual,
				Value:    "node1",
			},
			{
				Kind:     constraint.KindField,
				Target:   "request",
				Field:    []string{"source"},
				Operator: constraint.OpIn,
				Value:    []string{"console", "pluto"},
			},
		},
	},
}

// -----------------------------------------------------------------------------
// Scenario: User can't rename sy_node_1_metrics channels
// Type: Security
// Status: SOLVED - uses KindRelationship on resource.owner
// -----------------------------------------------------------------------------

// DenyRenameNodeChannels prevents renaming channels owned by nodes.
var DenyRenameNodeChannels = policy.Policy{
	Key:    uuid.New(),
	Name:   "deny-rename-node-channels",
	Effect: policy.EffectDeny,
	Constraint: constraint.Constraint{
		Kind:            constraint.KindRelationship,
		Objects:         []ontology.ID{{Type: "channel"}},
		Actions:         []access.Action{access.ActionUpdate},
		Relationship:    ontology.CreatedBy,
		Operator:        constraint.OpContainsAny,
		RelationshipIDs: []ontology.ID{{Type: cluster.OntologyTypeNode}},
	},
}

// -----------------------------------------------------------------------------
// Scenario: User wants to "lock" a workspace to only be edited by them
// Type: Security (ReBAC)
// Status: SOLVED - uses KindRelationship for created_by
// -----------------------------------------------------------------------------

// LockedWorkspaceCreatorOnly allows only the creator to edit a locked workspace.
var LockedWorkspaceCreatorOnly = policy.Policy{
	Key:    uuid.New(),
	Name:   "locked-workspace-creator-only",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:    constraint.KindAnd,
		Objects: []ontology.ID{{Type: "workspace"}},
		Actions: []access.Action{access.ActionUpdate, access.ActionDelete},
		Constraints: []constraint.Constraint{
			{
				Kind:     constraint.KindField,
				Target:   "resource",
				Field:    []string{"locked"},
				Operator: constraint.OpEqual,
				Value:    true,
			},
			{
				Kind:         constraint.KindRelationship,
				Relationship: "created_by",
				Operator:     constraint.OpContainsAny,
				MatchSubject: true,
			},
		},
	},
}

// =============================================================================
// UX SCENARIOS FROM TABLE (for filtering, not hard security)
// =============================================================================

// -----------------------------------------------------------------------------
// Scenario: User can retrieve sy_task_set via Pluto, but not via Console
// Type: UX
// Status: SOLVED - uses KindField on request.source
// -----------------------------------------------------------------------------

// AllowTaskSetFromPlutoOnly allows retrieving task set only from Pluto client.
var AllowTaskSetFromPlutoOnly = policy.Policy{
	Key:    uuid.New(),
	Name:   "allow-task-set-from-pluto-only",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:     constraint.KindField,
		Objects:  []ontology.ID{{Type: "channel", Key: "sy_task_set"}},
		Actions:  []access.Action{access.ActionRetrieve},
		Target:   "request",
		Field:    []string{"source"},
		Operator: constraint.OpEqual,
		Value:    "pluto",
	},
}

// -----------------------------------------------------------------------------
// Scenario: Task statuses can be updated via Driver, not Console
// Type: UX
// Status: SOLVED - uses KindField on request.source
// -----------------------------------------------------------------------------

// AllowTaskStatusUpdateFromDriver allows task status updates only from Driver.
var AllowTaskStatusUpdateFromDriver = policy.Policy{
	Key:    uuid.New(),
	Name:   "allow-task-status-update-from-driver",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:     constraint.KindField,
		Objects:  []ontology.ID{{Type: "status"}},
		Actions:  []access.Action{access.ActionUpdate},
		Target:   "request",
		Field:    []string{"source"},
		Operator: constraint.OpEqual,
		Value:    "driver",
	},
}

// -----------------------------------------------------------------------------
// Scenario: User can see sy_node_1_metrics but not sy_task_cmd in Console
// Type: UX
// Status: SOLVED - uses KindRelationship on resource.owner_type
// -----------------------------------------------------------------------------

// AllowUserVisibleChannels allows viewing channels not owned by builtin/system.
var AllowUserVisibleChannels = policy.Policy{
	Key:    uuid.New(),
	Name:   "allow-user-visible-channels",
	Effect: policy.EffectAllow,
	Constraint: constraint.Constraint{
		Kind:            constraint.KindRelationship,
		Objects:         []ontology.ID{{Type: "channel"}},
		Actions:         []access.Action{access.ActionRetrieve},
		Relationship:    ontology.CreatedBy,
		Operator:        constraint.OpContainsNone,
		RelationshipIDs: []ontology.ID{{Type: ontology.TypeBuiltIn}, {Type: cluster.OntologyTypeNode}},
	},
}

// =============================================================================
// COMPUTED CONSTRAINT EXAMPLES
// =============================================================================

// MaxDurationDataRead limits data reads to a maximum duration.
// Uses KindComputed to check the duration of the time range.
func MaxDurationDataRead(maxDuration telem.TimeSpan) policy.Policy {
	return policy.Policy{
		Key:    uuid.New(),
		Name:   "max-duration-data-read",
		Effect: policy.EffectAllow,
		Constraint: constraint.Constraint{
			Kind:     constraint.KindComputed,
			Objects:  []ontology.ID{{Type: "channel"}},
			Actions:  []access.Action{access.ActionRetrieve},
			Property: "duration",
			Source:   []string{"request", "time_range"},
			Operator: constraint.OpLessThanOrEqual,
			Value:    maxDuration,
		},
	}
}

// =============================================================================
// SUMMARY OF CONSTRAINT KINDS
// =============================================================================
//
// KindField:
//   Target: "resource" | "subject" | "request" | "system"
//   Field:  []string path to the field (e.g., ["status"], ["clearance"])
//   Operators: OpEqual, OpNotEqual, OpIn, OpNotIn, OpContains, OpContainsAny,
//              OpContainsAll, OpContainsNone, OpWithin, OpSubsetOf,
//              OpLessThan, OpLessThanOrEqual, OpGreaterThan, OpGreaterThanOrEqual
//
// KindRelationship:
//   Relationship: "created_by" | "labeled_by" | "parent_of" | "member_of" | etc.
//   Operators: OpContainsAny, OpContainsAll, OpContainsNone
//   RelationshipIDs: []ontology.ID - the IDs to match against
//   MatchSubject: bool - when true, matches against the requesting subject instead of RelationshipIDs
//
// KindComputed:
//   Property: "duration" | "age" | "count"
//   Source:   []string path to source value
//   Operators: comparison operators (lt, lte, gt, gte, eq)
//
// KindAnd:
//   Constraints: []Constraint - all must be satisfied
//
// KindOr:
//   Constraints: []Constraint - at least one must be satisfied
//
// KindNot:
//   Constraints: []Constraint - first element is inverted
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
