// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package inspect

import (
	"fmt"

	"github.com/synnaxlabs/cesium/internal/channel"
)

// Severity classifies how bad a finding is.
type Severity string

const (
	// SeverityError marks a broken storage invariant.
	SeverityError Severity = "error"
	// SeverityWarning marks a likely bug or degradation the engine tolerates.
	SeverityWarning Severity = "warning"
	// SeverityInfo marks a notable observation that is not wrong.
	SeverityInfo Severity = "info"
)

// Check names one inspection with a stable identifier.
type Check string

const (
	// CheckIndexDecode reports an index or counter file that cannot be decoded.
	CheckIndexDecode Check = "cesium.index-decode"
	// CheckDomainOrder reports domains that are not sorted by start timestamp.
	CheckDomainOrder Check = "cesium.domain-order"
	// CheckDomainOverlap reports domains that overlap a neighbor.
	CheckDomainOverlap Check = "cesium.domain-overlap"
	// CheckDomainBounds reports domains whose time range is invalid.
	CheckDomainBounds Check = "cesium.domain-bounds"
	// CheckFileKey reports domains referencing file key zero or a key above the
	// allocated counter.
	CheckFileKey Check = "cesium.file-key"
	// CheckFileBounds reports domains whose offset plus size exceeds the data file.
	CheckFileBounds Check = "cesium.file-bounds"
	// CheckMissingFile reports domains referencing a data file that does not exist.
	CheckMissingFile Check = "cesium.missing-file"
	// CheckDensityAlign reports fixed-density domains whose size is not a multiple of
	// the channel's density.
	CheckDensityAlign Check = "cesium.density-align"
	// CheckMeta reports a missing, undecodable, or invalid meta.json.
	CheckMeta Check = "cesium.meta"
	// CheckOrphanFile reports files no pointer references and unrecognized files.
	CheckOrphanFile Check = "cesium.orphan-file"
	// CheckArtifacts reports leftovers from an interrupted GC or delete.
	CheckArtifacts Check = "cesium.artifacts"
	// CheckIgnored reports channels the engine silently ignores on open.
	CheckIgnored Check = "cesium.ignored"
	// CheckIndexRef reports data channels whose index channel is absent or invalid.
	CheckIndexRef Check = "cesium.index-ref"
	// CheckGarbage reports channels whose garbage ratio shows GC is not keeping up.
	CheckGarbage Check = "cesium.garbage"
	// CheckTinyDomain reports channels dominated by abnormally small domains.
	CheckTinyDomain Check = "cesium.tiny-domain"
	// CheckMicroGap reports near-zero gaps between domains that should be continuous.
	CheckMicroGap Check = "cesium.micro-gap"
	// CheckTimeBounds reports domain bounds implausibly far in the past or future.
	CheckTimeBounds Check = "cesium.time-bounds"
	// CheckIndexContent reports index channels whose stored timestamps disagree with
	// their domain bounds. Deep.
	CheckIndexContent Check = "cesium.index-content"
	// CheckVarLenWalk reports variable-length domains whose length prefixes do not
	// walk cleanly to the domain's end. Deep.
	CheckVarLenWalk Check = "cesium.varlen-walk"
)

// severities fixes the severity of every check.
var severities = map[Check]Severity{
	CheckIndexDecode:   SeverityError,
	CheckDomainOrder:   SeverityError,
	CheckDomainOverlap: SeverityError,
	CheckDomainBounds:  SeverityError,
	CheckFileKey:       SeverityError,
	CheckFileBounds:    SeverityError,
	CheckMissingFile:   SeverityError,
	CheckDensityAlign:  SeverityError,
	CheckMeta:          SeverityError,
	CheckOrphanFile:    SeverityWarning,
	CheckArtifacts:     SeverityWarning,
	CheckIgnored:       SeverityWarning,
	CheckIndexRef:      SeverityError,
	CheckGarbage:       SeverityWarning,
	CheckTinyDomain:    SeverityWarning,
	CheckMicroGap:      SeverityWarning,
	CheckTimeBounds:    SeverityWarning,
	CheckIndexContent:  SeverityError,
	CheckVarLenWalk:    SeverityError,
}

// hints fixes the remediation hint of every check.
var hints = map[Check]string{
	CheckIndexDecode:   "the index file is corrupt; restore the channel from backup",
	CheckDomainOrder:   "the index is corrupt; reads may return wrong data",
	CheckDomainOverlap: "the index is corrupt; reads may return wrong data",
	CheckDomainBounds:  "the index is corrupt; reads may return wrong data",
	CheckFileKey:       "the index references unallocated files; data may be missing",
	CheckFileBounds:    "the data file is truncated below the recorded domain size",
	CheckMissingFile:   "a data file was lost; reads in its domains will fail",
	CheckDensityAlign:  "the domain holds a partial sample; the tail may be corrupt",
	CheckMeta:          "the channel cannot be opened; inspect meta.json by hand",
	CheckOrphanFile:    "the file is unreachable and only consumes disk",
	CheckArtifacts:     "an interrupted GC or delete left temporaries; safe to remove",
	CheckIgnored:       "the engine skips this channel on open; its disk is wasted",
	CheckIndexRef:      "data cannot be read without its index channel",
	CheckGarbage:       "garbage exceeds the GC threshold; check GC configuration",
	CheckTinyDomain:    "many tiny domains slow reads; check writer commit patterns",
	CheckMicroGap:      "writers leave near-zero gaps; domains never merge",
	CheckTimeBounds:    "timestamps are implausible; check what wrote this range",
	CheckIndexContent:  "index samples disagree with domain bounds; reads misalign",
	CheckVarLenWalk:    "sample framing is corrupt; reads in the domain will fail",
}

// Finding is one observed problem.
type Finding struct {
	// Check identifies the inspection that produced the finding.
	Check Check `json:"check"`
	// Severity is the fixed severity of the check.
	Severity Severity `json:"severity"`
	// Channel is the key of the channel the finding is about; zero for findings
	// about the database root.
	Channel channel.Key `json:"channel"`
	// Subject narrows the finding within the channel: a file name or a domain
	// position.
	Subject string `json:"subject"`
	// Message describes what was observed.
	Message string `json:"message"`
	// Hint is a short remediation pointer.
	Hint string `json:"hint"`
}

// newFinding builds a finding for the given check, filling in its fixed severity and
// hint.
func newFinding(
	check Check,
	ch channel.Key,
	subject string,
	format string,
	args ...any,
) Finding {
	return Finding{
		Check:    check,
		Severity: severities[check],
		Channel:  ch,
		Subject:  subject,
		Message:  fmt.Sprintf(format, args...),
		Hint:     hints[check],
	}
}
