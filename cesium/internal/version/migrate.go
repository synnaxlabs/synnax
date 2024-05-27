// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt

package version

import (
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"os"
	"strconv"
)

var migrations = map[string]func(fs xfs.FS) error{
	"12": migrate12,
	"02": migrate12,
}

func Migrate(fs xfs.FS, oldVersion Version, newVersion Version) error {
	migrate, ok := migrations[strconv.Itoa(int(oldVersion))+strconv.Itoa(int(newVersion))]
	if !ok {
		return errors.Newf("migration from version %d to version %d not found", oldVersion, newVersion)
	}
	err := migrate(fs)
	if err != nil {
		return errors.Wrap(err, "version migration error")
	}

	return nil
}

// migrate12 migrates the file version from V1 to V2. This involves prepending the
// index.domain file with 4 extra bytes at the start, indicating the number of pointers
// stored in the file. In addition, a tombstone.domain file is created.
func migrate12(fs xfs.FS) error {
	// First create a new file for tombstones
	f, err := fs.Open("tombstone.domain", os.O_CREATE|os.O_EXCL)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	// Then migrate the pointers
	r, err := fs.Open("index.domain", os.O_RDONLY)
	if err != nil {
		return err
	}
	w, err := fs.Open("index.domain", os.O_WRONLY)
	if err != nil {
		return err
	}

	s, err := r.Stat()
	if err != nil {
		return err
	}
	var (
		sz  = make([]byte, 4)
		buf = make([]byte, 26)
	)

	for i := s.Size() - 26; i >= 0; i -= 26 {
		_, err = r.ReadAt(buf, i)
		if err != nil {
			return err
		}

		_, err = w.WriteAt(buf, i+4)
		if err != nil {
			return err
		}
	}

	telem.ByteOrder.PutUint32(sz, uint32(s.Size()/26))
	_, err = w.WriteAt(sz, 0)
	if err != nil {
		return err
	}

	err = r.Close()
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return nil
}
