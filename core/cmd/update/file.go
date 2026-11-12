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

// fromFile replaces the installed Core with the Core binary at file, as apply does.
func fromFile(ctx context.Context, w io.Writer, file string, timeout time.Duration) (
	err error,
) {
	inst, installed, err := installation()
	if err != nil {
		return err
	}
	staged := inst.Binary + stagedSuffix
	defer func() { err = errors.Combine(err, removeStaged(staged)) }()
	if err = copyBinary(file, staged); err != nil {
		return err
	}
	to, err := binaryVersion(ctx, staged)
	if err != nil {
		return errors.Wrapf(err, "%s is not a Core binary for this computer", file)
	}
	cur, ok, err := check(ctx, w, inst, installed, to)
	if !ok || err != nil {
		return err
	}
	return cur.apply(ctx, w, staged, to, timeout)
}

// current is the installed Core that an update replaces.
type current struct {
	binary  string
	version string
	// backups holds the backups of the data of the Core service, or is nil when no
	// service runs the Core.
	backups *storage.Backups
}

// check returns the installed Core when it can update to Core to. It returns false,
// after it prints why, when Core to is already installed.
func check(
	ctx context.Context,
	w io.Writer,
	inst service.Installation,
	installed bool,
	to string,
) (current, bool, error) {
	from, err := binaryVersion(ctx, inst.Binary)
	if err != nil {
		return current{}, false, err
	}
	cur := current{binary: inst.Binary, version: from}
	if version.Compare(to, from) < 0 {
		return current{}, false, errors.Newf(
			"Core %s is older than the installed Core %s. To go back, run `synnax "+
				"update undo`.",
			to,
			from,
		)
	}
	if !installed {
		if to == from {
			return current{}, false, printLines(
				w,
				fmt.Sprintf("Core %s is already installed at %s.", to, inst.Binary),
			)
		}
		return cur, true, nil
	}
	if cur.backups, err = storage.NewBackups(
		storage.BackupsConfig{Dirname: inst.Data},
	); err != nil {
		return current{}, false, errors.Wrapf(
			err,
			"failed to open the data of the Core service",
		)
	}
	state, err := cur.backups.UpdateState(to)
	if err != nil {
		return current{}, false, err
	}
	switch {
	case state == storage.UpdateUndone:
		return current{}, false, errors.Newf(
			"The update to Core %s failed before. Fix the cause, run `synnax update "+
				"retry`, then update again.",
			to,
		)
	// The pending update still needs the old binary that a replace would overwrite.
	case state == storage.UpdatePending:
		return current{}, false, errors.New(
			"An update of the data did not finish. Wait for the Core service to " +
				"finish it, and start the service if it stopped.",
		)
	case to == from:
		return current{}, false, printLines(
			w,
			fmt.Sprintf("Core %s is already installed.", to),
		)
	}
	return cur, true, nil
}

// apply replaces the installed Core with the Core binary at staged, which is Core to.
// With a Core service, it stops the service, replaces the binary, starts the service,
// and waits up to timeout for the new Core to commit or undo the update. When the new
// Core undoes the update, apply puts the old binary back and starts the service again.
func (c current) apply(
	ctx context.Context,
	w io.Writer,
	staged, to string,
	timeout time.Duration,
) error {
	replaced := fmt.Sprintf(
		"Replaced Core %s with Core %s at %s.",
		c.version,
		to,
		c.binary,
	)
	if c.backups == nil {
		if err := replace(c.binary, staged); err != nil {
			return err
		}
		return printLines(
			w,
			replaced,
			"Start the Core. It backs up the data, updates it, and undoes the "+
				"update if it fails.",
		)
	}
	if err := printLines(w, "Stopping the Core service."); err != nil {
		return err
	}
	if err := service.Stop(); err != nil {
		return err
	}
	if err := replace(c.binary, staged); err != nil {
		return errors.Combine(err, service.Start())
	}
	if err := printLines(
		w,
		replaced,
		fmt.Sprintf("Starting the Core service. Waiting for Core %s to update the "+
			"data.", to),
	); err != nil {
		return err
	}
	if err := service.Start(); err != nil {
		return err
	}
	state, err := waitForUpdate(ctx, c.backups, to, timeout)
	if err != nil {
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
			c.version,
		)); err != nil {
			return err
		}
		if err = service.Stop(); err != nil {
			return err
		}
		if err = os.Rename(c.binary+oldSuffix, c.binary); err != nil {
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
			c.version,
		)
	default:
		return errors.Newf(
			"Core %s did not finish the update within %s. Check the Core log. If the "+
				"Core undoes the update, run `synnax update undo` to put back Core %s.",
			to,
			timeout,
			c.version,
		)
	}
}

// removeStaged removes the staged binary, if it exists.
func removeStaged(staged string) error {
	if err := os.Remove(staged); !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
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
	if v, err := oldVersion(ctx, binary); err != nil || v != want {
		return false, err
	}
	old, swap := binary+oldSuffix, binary+".swap"
	if err := os.Rename(binary, swap); err != nil {
		return false, err
	}
	if err := os.Rename(old, binary); err != nil {
		return false, errors.Combine(err, os.Rename(swap, binary))
	}
	return true, os.Rename(swap, old)
}

// oldVersion returns the Core version of the binary that an update replaced, or an
// empty string when there is none.
func oldVersion(ctx context.Context, binary string) (string, error) {
	old := binary + oldSuffix
	if _, err := os.Stat(old); errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return binaryVersion(ctx, old)
}

func printLines(w io.Writer, lines ...string) error {
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
