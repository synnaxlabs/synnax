// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage

import (
	"context"
	"os"
	"strings"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"go.uber.org/zap"
)

const (
	// updateMarkerFile marks an update that did not commit. Its first line names the
	// backup that undoes the update, and its second line is the version of the update.
	updateMarkerFile = "UPDATING"
	// failedUpdateFile holds the version of the last update that failed.
	failedUpdateFile = "UPDATE_FAILED"
)

// Update is an update of the data to a newer Core version that did not commit yet.
type Update struct {
	// From is the version of the data before the update, or UnknownVersion.
	From string
	// To is the version of the Core that updates the data.
	To      string
	backup  string
	backups *Backups
}

// Commit keeps the updated data, so that a later failure no longer undoes the update.
func (u *Update) Commit() error { return u.backups.fs.Remove(updateMarkerFile) }

// Undo puts back the backup taken before the update and records the update as failed. A
// Core of version To then refuses the data until Backups.Retry. Undo returns an error
// that names cause and the Core version to run. When ctx is done, the start stopped on
// request, so Undo records no failure and returns cause, and the next start of version
// To updates the data again. Call Undo only after the Layer that started the update
// closes.
func (u *Update) Undo(ctx context.Context, cause error) error {
	if ctx.Err() != nil {
		if _, err := u.backups.Restore(u.backup); err != nil {
			return errors.Combine(
				errors.Wrapf(err, "failed to put back the data before Core %s", u.To),
				cause,
			)
		}
		u.backups.cfg.L.Warn(
			"the Core stopped before it committed the update, so it put back the "+
				"data. Its next start updates the data again.",
			zap.String("to", u.To),
		)
		return cause
	}
	if err := u.backups.undo(u.backup, u.To, u.backups.Restore); err != nil {
		return errors.Newf(
			"Update to Core %s failed: %v. The undo of the update also failed: %v",
			u.To,
			cause,
			err,
		)
	}
	return failedUpdateError(u.To, u.From, cause)
}

// undo records the update to version as failed, then puts back the backup with restore.
func (b *Backups) undo(
	backup, version string,
	restore func(string) (Restored, error),
) error {
	if err := xfs.WriteFile(b.fs, failedUpdateFile, []byte(version)); err != nil {
		return err
	}
	res, err := restore(backup)
	if err != nil {
		return err
	}
	b.cfg.L.Warn(
		"undid the update",
		zap.String("to", version),
		zap.String("backup", backup),
		zap.String("orphaned_dir", res.OrphanedDir),
	)
	return nil
}

// failedUpdateError returns the error for the failed update to version, with the data
// back at recorded. cause is the reason, or nil when it is not known.
func failedUpdateError(version, recorded string, cause error) error {
	failed := "Update to Core " + version + " failed"
	if cause != nil {
		failed += ": " + cause.Error()
	}
	run := "Run Core " + recorded
	back := "Core " + recorded
	if recorded == UnknownVersion {
		run, back = "Run that version", "the version before the update"
	}
	return errors.Mark(errors.Newf(
		"%s. The data is back at %s. %s, or fix the cause and run `synnax update "+
			"retry`.",
		failed,
		back,
		run,
	), ErrRefused)
}

// checkUpdate undoes an update that an earlier start did not commit. It then refuses
// the data when current is the version of a failed update. The caller holds the lock on
// the data directory.
func (b *Backups) checkUpdate(current string) error {
	marker, err := xfs.ReadFile(b.fs, updateMarkerFile)
	if err == nil {
		backup, version, _ := strings.Cut(string(marker), "\n")
		b.cfg.L.Warn(
			"the last start stopped before it committed the update, so this start "+
				"undoes it",
			zap.String("to", version),
		)
		if err = b.undo(backup, version, b.restore); err != nil {
			return errors.Wrapf(err, "failed to undo the update to Core %s", version)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	failed, err := b.failedVersion()
	if err != nil || failed != current {
		return err
	}
	recorded, err := b.readVersion()
	if err != nil {
		return err
	}
	return failedUpdateError(current, recorded, nil)
}

// failedVersion returns the version of the last failed update, or an empty string when
// none failed.
func (b *Backups) failedVersion() (string, error) {
	data, err := xfs.ReadFile(b.fs, failedUpdateFile)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(data), err
}

// Retry removes the record of the failed update, so that its Core version can update
// the data again. It returns the version of the update, and an error when no update
// failed.
func (b *Backups) Retry() (string, error) {
	failed, err := b.failedVersion()
	if err != nil {
		return "", err
	}
	if failed == "" {
		return "", errors.New("there is no failed update to retry")
	}
	return failed, b.fs.Remove(failedUpdateFile)
}

// UpdateState is the progress of an update of the data to a Core version.
type UpdateState uint8

const (
	// UpdateNotStarted is an update that no Core started on the data.
	UpdateNotStarted UpdateState = iota
	// UpdatePending is an update that started and did not commit or undo yet.
	UpdatePending
	// UpdateCommitted is an update that the Core committed.
	UpdateCommitted
	// UpdateUndone is an update that failed. The data is back at the version before it.
	UpdateUndone
)

// UpdateState returns the state of the update of the data to version. It takes no lock,
// so it can watch a Core that runs on the data.
func (b *Backups) UpdateState(version string) (UpdateState, error) {
	// The Core writes the marker before the version, and a restore writes the version
	// before it removes the markers. A version that changed during the reads is part of
	// one of these steps.
	recorded, err := b.readVersion()
	if err != nil {
		return 0, err
	}
	updating, err := b.fs.Exists(updateMarkerFile)
	if err != nil {
		return 0, err
	}
	restoring, err := b.fs.Exists(restoreMarkerFile)
	if err != nil {
		return 0, err
	}
	failed, err := b.failedVersion()
	if err != nil {
		return 0, err
	}
	again, err := b.readVersion()
	if err != nil {
		return 0, err
	}
	switch {
	case updating || restoring || again != recorded:
		return UpdatePending, nil
	case failed == version:
		return UpdateUndone, nil
	case recorded == version:
		return UpdateCommitted, nil
	}
	return UpdateNotStarted, nil
}
