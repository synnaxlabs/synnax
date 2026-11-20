// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package tz resolves IANA time zone names against a zone database embedded in the
// binary. x/cpp/tz reads the same database.
package tz

//go:generate go run gen.go

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Database resolves zone names to locations. Names match without regard to case. A
// Database is not safe for concurrent use.
type Database struct {
	// files maps each lowercase zone name to its entry in the database.
	files map[string]*zip.File
	// locations caches each zone already loaded, by lowercase name.
	locations map[string]*time.Location
}

// Open reads the embedded zone database.
func Open() (*Database, error) {
	data, err := base64.StdEncoding.DecodeString(zoneinfo)
	if err != nil {
		return nil, err
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	db := &Database{
		files:     make(map[string]*zip.File, len(r.File)),
		locations: make(map[string]*time.Location),
	}
	for _, f := range r.File {
		db.files[strings.ToLower(f.Name)] = f
	}
	return db, nil
}

// Load returns the location for the zone name, matched without regard to case. It
// returns query.ErrNotFound when the database has no zone with that name.
func (d *Database) Load(name string) (*time.Location, error) {
	key := strings.ToLower(name)
	if loc, ok := d.locations[key]; ok {
		return loc, nil
	}
	f, ok := d.files[key]
	if !ok {
		return nil, errors.Wrapf(query.ErrNotFound, "time zone %q", name)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(rc)
	if closeErr := rc.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocationFromTZData(f.Name, data)
	if err != nil {
		return nil, err
	}
	d.locations[key] = loc
	return loc, nil
}

// Names returns every zone name in the database, sorted.
func (d *Database) Names() []string {
	names := make([]string, 0, len(d.files))
	for _, f := range d.files {
		names = append(names, f.Name)
	}
	slices.Sort(names)
	return names
}
