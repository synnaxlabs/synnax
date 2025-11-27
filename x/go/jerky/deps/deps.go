// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package deps provides dependency tracking for jerky-managed types.
package deps

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/synnaxlabs/x/jerky/state"
)

// Registry tracks all jerky-managed types and their dependencies.
type Registry struct {
	// types maps fully qualified type name (pkg/path.TypeName) to its state file location
	types map[string]TypeInfo
}

// TypeInfo contains information about a jerky-managed type.
type TypeInfo struct {
	// PackagePath is the Go import path
	PackagePath string
	// TypeName is the struct name
	TypeName string
	// StateDir is the directory containing the jerky.state.json file
	StateDir string
	// CurrentVersion is the current version of this type
	CurrentVersion int
	// CompositeHash is the hash including dependencies
	CompositeHash string
}

// NewRegistry creates a new dependency registry.
func NewRegistry() *Registry {
	return &Registry{
		types: make(map[string]TypeInfo),
	}
}

// Register adds a type to the registry.
func (r *Registry) Register(info TypeInfo) {
	key := fmt.Sprintf("%s.%s", info.PackagePath, info.TypeName)
	r.types[key] = info
}

// Get retrieves type info by fully qualified name.
func (r *Registry) Get(qualifiedName string) (TypeInfo, bool) {
	info, ok := r.types[qualifiedName]
	return info, ok
}

// GetByPackageAndType retrieves type info by package path and type name.
func (r *Registry) GetByPackageAndType(packagePath, typeName string) (TypeInfo, bool) {
	key := fmt.Sprintf("%s.%s", packagePath, typeName)
	return r.Get(key)
}

// LoadFromStateFile loads type info from a state file directory.
func (r *Registry) LoadFromStateFile(stateDir string) error {
	stateFile, err := state.Load(stateDir)
	if err != nil {
		return err
	}

	for typeName, typeState := range stateFile.Types {
		latest := typeState.LatestVersion()
		if latest == nil {
			continue
		}
		r.Register(TypeInfo{
			PackagePath:    typeState.Package,
			TypeName:       typeName,
			StateDir:       stateDir,
			CurrentVersion: typeState.CurrentVersion,
			CompositeHash:  latest.CompositeHash,
		})
	}
	return nil
}

// Dependency represents a dependency on another jerky-managed type.
type Dependency struct {
	PackagePath string
	TypeName    string
	Version     int
	Hash        string
}

// ResolveDependencies finds all jerky-managed dependencies of a type.
func (r *Registry) ResolveDependencies(deps []string) ([]Dependency, error) {
	var resolved []Dependency
	for _, dep := range deps {
		if info, ok := r.Get(dep); ok {
			resolved = append(resolved, Dependency{
				PackagePath: info.PackagePath,
				TypeName:    info.TypeName,
				Version:     info.CurrentVersion,
				Hash:        info.CompositeHash,
			})
		}
	}
	return resolved, nil
}

// DiscoverJerkyTypes scans a directory tree for jerky state files and registers all types.
func (r *Registry) DiscoverJerkyTypes(rootDir string) error {
	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == state.StateFileName {
			dir := filepath.Dir(path)
			return r.LoadFromStateFile(dir)
		}
		return nil
	})
}

// CheckForCycles detects circular dependencies in the type graph.
// Returns an error if a cycle is detected, with the cycle path.
func (r *Registry) CheckForCycles(startType string, getDeps func(string) []string) error {
	visited := make(map[string]bool)
	path := make([]string, 0)

	var visit func(string) error
	visit = func(typeName string) error {
		if visited[typeName] {
			// Check if it's in current path (cycle)
			for i, p := range path {
				if p == typeName {
					cyclePath := append(path[i:], typeName)
					return fmt.Errorf("circular dependency detected: %v", cyclePath)
				}
			}
			return nil
		}

		visited[typeName] = true
		path = append(path, typeName)

		for _, dep := range getDeps(typeName) {
			if err := visit(dep); err != nil {
				return err
			}
		}

		path = path[:len(path)-1]
		return nil
	}

	return visit(startType)
}
