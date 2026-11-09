// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package update

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/synnaxlabs/synnax/cmd/service"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/errors"
)

const (
	// oldSuffix names the binary that an update replaced, next to the binary that
	// replaced it.
	oldSuffix = ".old"
	// stagedSuffix names the new binary while the update checks it.
	stagedSuffix = ".new"
	pollInterval = 250 * time.Millisecond
)

// fromFile replaces the installed Core with the Core binary at file. With a Core
// service, it stops the service, replaces the binary, starts the service, and waits up
// to timeout for the new Core to commit or undo the update. When the new Core undoes
// the update, fromFile puts the old binary back and starts the service again.
func fromFile(ctx context.Context, w io.Writer, file string, timeout time.Duration) (
	err error,
) {
	inst, installed, err := installation()
	if err != nil {
		return err
	}
	binary := inst.Binary
	staged := binary + stagedSuffix
	if err = copyBinary(file, staged); err != nil {
		return err
	}
	defer func() {
		if rmErr := os.Remove(staged); !errors.Is(rmErr, os.ErrNotExist) {
			err = errors.Combine(err, rmErr)
		}
	}()
	to, err := binaryVersion(ctx, staged)
	if err != nil {
		return errors.Wrapf(err, "%s is not a Core binary for this computer", file)
	}
	from, err := binaryVersion(ctx, binary)
	if err != nil {
		return err
	}
	if version.Compare(to, from) < 0 {
		return errors.Newf(
			"Core %s is older than the installed Core %s. To go back, run `synnax "+
				"update undo`.",
			to,
			from,
		)
	}
	if !installed {
		if to == from {
			return printLines(
				w,
				fmt.Sprintf("Core %s is already installed at %s.", to, binary),
			)
		}
		if err = replace(binary, staged); err != nil {
			return err
		}
		return printLines(
			w,
			fmt.Sprintf("Replaced Core %s with Core %s at %s.", from, to, binary),
			"Start the Core. It backs up the data, updates it, and undoes the "+
				"update if it fails.",
		)
	}
	backups, err := storage.NewBackups(storage.BackupsConfig{Dirname: inst.Data})
	if err != nil {
		return errors.Wrapf(err, "failed to open the data of the Core service")
	}
	state, err := backups.UpdateState(to)
	if err != nil {
		return err
	}
	switch {
	case state == storage.UpdateUndone:
		return errors.Newf(
			"The update to Core %s failed before. Fix the cause, run `synnax update "+
				"retry`, then update again.",
			to,
		)
	// The pending update still needs the old binary that a replace would overwrite.
	case state == storage.UpdatePending:
		return errors.New(
			"An update of the data did not finish. Wait for the Core service to " +
				"finish it, and start the service if it stopped.",
		)
	case to == from:
		return printLines(w, fmt.Sprintf("Core %s is already installed.", to))
	}
	if err = printLines(w, "Stopping the Core service."); err != nil {
		return err
	}
	if err = service.Stop(); err != nil {
		return err
	}
	if err = replace(binary, staged); err != nil {
		return errors.Combine(err, service.Start())
	}
	if err = printLines(
		w,
		fmt.Sprintf("Replaced Core %s with Core %s at %s.", from, to, binary),
		fmt.Sprintf("Starting the Core service. Waiting for Core %s to update the "+
			"data.", to),
	); err != nil {
		return err
	}
	if err = service.Start(); err != nil {
		return err
	}
	if state, err = waitForUpdate(ctx, backups, to, timeout); err != nil {
		return err
	}
	switch state {
	case storage.UpdateCommitted:
		return printLines(w, fmt.Sprintf("Updated to Core %s.", to))
	case storage.UpdateUndone:
		if err = printLines(w, fmt.Sprintf(
			"Core %s failed to update the data and put it back. Starting Core %s "+
				"again.",
			to,
			from,
		)); err != nil {
			return err
		}
		if err = service.Stop(); err != nil {
			return err
		}
		if err = os.Rename(binary+oldSuffix, binary); err != nil {
			return err
		}
		if err = service.Start(); err != nil {
			return err
		}
		return errors.Newf(
			"Update to Core %s failed. Core %s runs again. Find the cause in the "+
				"Core log, fix it, and run `synnax update retry` before you update "+
				"again.",
			to,
			from,
		)
	default:
		return errors.Newf(
			"Core %s did not finish the update within %s. Check the Core log.",
			to,
			timeout,
		)
	}
}

// installation returns the Core service on this host, and reports whether there is one.
// Without a service, the Binary of the result is the running binary.
func installation() (service.Installation, bool, error) {
	inst, installed, err := service.Lookup()
	if err != nil || installed {
		return inst, installed, err
	}
	exe, err := os.Executable()
	if err != nil {
		return inst, false, errors.Wrap(err, "failed to get executable path")
	}
	inst.Binary, err = filepath.EvalSymlinks(exe)
	return inst, false, err
}

// copyBinary copies the binary at src to dst, which anyone can run.
func copyBinary(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, in.Close()) }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		return errors.Combine(err, out.Close())
	}
	if err = out.Close(); err != nil {
		return err
	}
	// The umask can remove permission bits from the mode of a new file.
	return os.Chmod(dst, 0o755)
}

// binaryVersion returns the version of the Core binary at path.
func binaryVersion(ctx context.Context, path string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").Output()
	if err != nil {
		return "", errors.Wrapf(err, "failed to run %s", path)
	}
	fields := strings.Fields(string(out))
	if len(fields) < 2 || fields[0] != "Synnax" ||
		(!version.Valid(fields[1]) && !version.IsDev(fields[1])) {
		return "", errors.Newf("%s printed no Core version", path)
	}
	return fields[1], nil
}

// replace moves binary to binary.old, then moves staged to binary.
func replace(binary, staged string) error {
	old := binary + oldSuffix
	if err := os.Rename(binary, old); err != nil {
		return err
	}
	if err := os.Rename(staged, binary); err != nil {
		return errors.Combine(err, os.Rename(old, binary))
	}
	return nil
}

// waitForUpdate waits up to timeout for the Core to commit or undo the update of the
// data to version. When the timeout passes first, it returns the last state.
func waitForUpdate(
	ctx context.Context,
	backups *storage.Backups,
	version string,
	timeout time.Duration,
) (storage.UpdateState, error) {
	deadline := time.After(timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		state, err := backups.UpdateState(version)
		if err != nil {
			return 0, err
		}
		if state == storage.UpdateCommitted || state == storage.UpdateUndone {
			return state, nil
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-deadline:
			return state, nil
		case <-ticker.C:
		}
	}
}

// putBackOld swaps binary with the binary that an update replaced, when that binary is
// Core version want. It reports whether it swapped them.
func putBackOld(ctx context.Context, binary, want string) (bool, error) {
	old := binary + oldSuffix
	if _, err := os.Stat(old); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if v, err := binaryVersion(ctx, old); err != nil || v != want {
		return false, err
	}
	swap := binary + ".swap"
	if err := os.Rename(binary, swap); err != nil {
		return false, err
	}
	if err := os.Rename(old, binary); err != nil {
		return false, errors.Combine(err, os.Rename(swap, binary))
	}
	return true, os.Rename(swap, old)
}

func printLines(w io.Writer, lines ...string) error {
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
