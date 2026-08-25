// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"fmt"

	"github.com/synnaxlabs/cesium/inspect"
)

// Severity classifies how bad a finding is.
type Severity = inspect.Severity

const (
	// SeverityError marks a broken storage invariant.
	SeverityError = inspect.SeverityError
	// SeverityWarning marks a likely bug or degradation the Core tolerates.
	SeverityWarning = inspect.SeverityWarning
	// SeverityInfo marks a notable observation that is not wrong.
	SeverityInfo = inspect.SeverityInfo
)

// Check names one inspection with a stable identifier.
type Check string

const (
	// CheckDecode reports a stored entry that no codec can decode.
	CheckDecode Check = "kv.decode"
	// CheckUnknownPrefix reports keys outside every known bucket.
	CheckUnknownPrefix Check = "kv.unknown-prefix"
	// CheckMigrationState reports a table behind the migration chain the binary
	// ships.
	CheckMigrationState Check = "kv.migration-state"
	// CheckCounter reports a key counter that would re-issue an existing key.
	CheckCounter Check = "kv.counter"
	// CheckStaging reports leftover entries from a partially-run migration.
	CheckStaging Check = "kv.staging"
	// CheckRelationshipKey reports a relationship key that cannot be parsed.
	CheckRelationshipKey Check = "ontology.rel-key"
	// CheckRelationshipEndpoint reports a relationship pointing at a resource that
	// no longer exists.
	CheckRelationshipEndpoint Check = "ontology.rel-endpoint"
	// CheckResourceType reports a resource whose type is not a known ontology type.
	CheckResourceType Check = "ontology.resource-type"
	// CheckResourceOrphan reports a resource whose backing entity is gone.
	CheckResourceOrphan Check = "ontology.resource-orphan"
	// CheckAlias reports a range alias whose range or channel is gone.
	CheckAlias Check = "ref.alias"
	// CheckRangeKV reports a range key-value pair whose range is gone.
	CheckRangeKV Check = "ref.range-kv"
	// CheckTaskConfig reports a config record with no task, or a task whose type has
	// no config store.
	CheckTaskConfig Check = "ref.task-config"
	// CheckCredentials reports credentials whose user no longer exists.
	CheckCredentials Check = "ref.credentials"
	// CheckRack reports a task or device referencing a deleted rack.
	CheckRack Check = "ref.rack"
	// CheckChannelIndex reports a channel whose index channel entry is gone.
	CheckChannelIndex Check = "ref.channel-index"
	// CheckPolicyObject reports a policy object referencing a deleted resource.
	CheckPolicyObject Check = "ref.policy-object"
	// CheckPanelTab reports a panel tab referencing a deleted entity.
	CheckPanelTab Check = "ref.panel-tab"
	// CheckChannelDir reports disagreement over which channels have Cesium storage.
	CheckChannelDir Check = "cross.channel-dir"
	// CheckChannelMeta reports disagreement between a channel entry and its
	// Cesium metadata.
	CheckChannelMeta Check = "cross.channel-meta"
)

// severities fixes the severity of every check.
var severities = map[Check]Severity{
	CheckDecode:               SeverityError,
	CheckUnknownPrefix:        SeverityWarning,
	CheckMigrationState:       SeverityWarning,
	CheckCounter:              SeverityError,
	CheckStaging:              SeverityWarning,
	CheckRelationshipKey:      SeverityError,
	CheckRelationshipEndpoint: SeverityWarning,
	CheckResourceType:         SeverityError,
	CheckResourceOrphan:       SeverityWarning,
	CheckAlias:                SeverityWarning,
	CheckRangeKV:              SeverityWarning,
	CheckTaskConfig:           SeverityWarning,
	CheckCredentials:          SeverityWarning,
	CheckRack:                 SeverityWarning,
	CheckChannelIndex:         SeverityError,
	CheckPolicyObject:         SeverityInfo,
	CheckPanelTab:             SeverityInfo,
	CheckChannelDir:           SeverityWarning,
	CheckChannelMeta:          SeverityError,
}

// hints fixes the remediation hint of every check.
var hints = map[Check]string{
	CheckDecode:               "the entry is unreadable; the Core drops it on every query",
	CheckUnknownPrefix:        "keys belong to no table; a migration may be unfinished",
	CheckMigrationState:       "start a Core against this directory once to migrate",
	CheckCounter:              "the next allocation would re-issue a key already in use",
	CheckStaging:              "a migration left staging entries; safe to remove",
	CheckRelationshipKey:      "the relationship is unreachable; ontology walks skip it",
	CheckRelationshipEndpoint: "the edge dangles; traversals return fewer results",
	CheckResourceType:         "ontology traversal fails on an unknown resource type",
	CheckResourceOrphan:       "the resource has no entity; queries silently drop it",
	CheckAlias:                "the alias resolves to nothing and cannot be deleted",
	CheckRangeKV:              "the pair is unreachable through its range",
	CheckTaskConfig:           "the task cannot be configured or its record is leaked",
	CheckCredentials:          "the login still works but has no user record",
	CheckRack:                 "the entity cannot be deployed until its rack exists",
	CheckChannelIndex:         "data cannot be read without its index channel",
	CheckPolicyObject:         "the policy grants access to nothing",
	CheckPanelTab:             "the tab renders empty for every user of the panel",
	CheckChannelDir:           "storage and metadata disagree on which channels exist",
	CheckChannelMeta:          "the Core and Cesium disagree; reads may fail or misalign",
}

// Finding is one observed problem.
type Finding struct {
	// Check identifies the inspection that produced the finding.
	Check Check `json:"check"`
	// Severity is the fixed severity of the check.
	Severity Severity `json:"severity"`
	// Subject narrows the finding: a table name, a key, or an entity identifier.
	Subject string `json:"subject"`
	// Message describes what was observed.
	Message string `json:"message"`
	// Hint is a short remediation pointer.
	Hint string `json:"hint"`
}

// newFinding builds a finding for the given check, filling in its fixed severity and
// hint.
func newFinding(check Check, subject, format string, args ...any) Finding {
	return Finding{
		Check:    check,
		Severity: severities[check],
		Subject:  subject,
		Message:  fmt.Sprintf(format, args...),
		Hint:     hints[check],
	}
}

// violation accumulates repeats of one check so a broken store yields one finding per
// check instead of one per entry.
type violation struct {
	// count is the number of entries that failed the check.
	count int
	// first is the subject of the first failure, used as the example.
	first string
}

// note records one failure with the given subject.
func (v *violation) note(subject string) {
	if v.count == 0 {
		v.first = subject
	}
	v.count++
}

// message renders label with the number of times the condition repeated.
func (v *violation) message(label string) string {
	if v.count < 2 {
		return label
	}
	return fmt.Sprintf("%s (%d occurrences)", label, v.count)
}
