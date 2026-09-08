// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package includes tracks the system and internal #include lines a generated
// C++ file needs, deduplicated in first-added order.
package includes

import "github.com/samber/lo"

// Manager accumulates a generated file's include lines. Template data embeds it so
// templates read HasIncludes/SystemIncludes/InternalIncludes directly.
type Manager struct {
	system   []string
	internal []string
}

// NewManager returns an empty Manager.
func NewManager() *Manager { return &Manager{} }

// AddSystem records a <system> include once.
func (m *Manager) AddSystem(name string) {
	if !lo.Contains(m.system, name) {
		m.system = append(m.system, name)
	}
}

// AddInternal records a repo-relative "internal" include once.
func (m *Manager) AddInternal(path string) {
	if !lo.Contains(m.internal, path) {
		m.internal = append(m.internal, path)
	}
}

// HasIncludes reports whether any include line was recorded.
func (m *Manager) HasIncludes() bool {
	return len(m.system) > 0 || len(m.internal) > 0
}

// SystemIncludes returns the recorded system includes in first-added order.
func (m *Manager) SystemIncludes() []string { return m.system }

// InternalIncludes returns the recorded internal includes in first-added order.
func (m *Manager) InternalIncludes() []string { return m.internal }
