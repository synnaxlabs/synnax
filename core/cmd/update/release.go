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
	"runtime"
	"time"

	"github.com/synnaxlabs/synnax/pkg/release"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// fromRelease downloads Core version to for this computer from the release, checks it
// against its signature, and replaces the installed Core with it, as apply does.
func fromRelease(
	ctx context.Context,
	w io.Writer,
	cfg release.Config,
	to string,
	timeout time.Duration,
) (err error) {
	if !version.Valid(to) {
		return errors.Mark(
			errors.Newf("%s is not a Core version, such as 0.61.0.", to),
			validate.ErrValidation,
		)
	}
	inst, installed, err := installation()
	if err != nil {
		return err
	}
	cur, ok, err := check(ctx, w, inst, installed, to)
	if !ok || err != nil {
		return err
	}
	file := release.File{Product: release.Core, Version: to, Platform: hostPlatform()}
	name := file.Name()
	if err = printLines(w, fmt.Sprintf("Downloading Core %s.", to)); err != nil {
		return err
	}
	staged := inst.Binary + stagedSuffix
	defer func() { err = errors.Combine(err, removeStaged(staged)) }()
	if err = release.Download(ctx, cfg, file, staged); err != nil {
		if errors.Is(err, release.ErrNotFound) {
			return errors.Newf(
				"Core %s has no signed release file %s. Check the version.",
				to,
				name,
			)
		}
		return err
	}
	if err = os.Chmod(staged, 0o755); err != nil {
		return err
	}
	got, err := binaryVersion(ctx, staged)
	if err != nil {
		return errors.Wrapf(
			err,
			"The release file %s does not run on this computer",
			name,
		)
	}
	if got != to {
		return errors.Newf("The release file %s holds Core %s.", name, got)
	}
	return cur.apply(ctx, w, staged, to, timeout)
}

// hostPlatform returns the platform of this computer.
func hostPlatform() release.Platform {
	if runtime.GOOS == "darwin" {
		return release.MacOS
	}
	return release.Platform(runtime.GOOS)
}
