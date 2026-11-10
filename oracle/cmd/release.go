// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/x/errors"
)

// stableTagRe matches a stable Core release tag. A release candidate carries a suffix.
var stableTagRe = regexp.MustCompile(`^core/v[0-9]+\.[0-9]+\.[0-9]+$`)

// latestStableTag returns the highest stable Core release tag in the repository.
func latestStableTag(ctx context.Context, repoRoot string) (string, error) {
	out, err := exec.CommandContext(
		ctx, "git", "-C", repoRoot, "tag", "--list", "core/v*", "--sort=-v:refname",
	).Output()
	if err != nil {
		return "", errors.Wrap(err, "list the Core release tags")
	}
	for tag := range strings.Lines(string(out)) {
		if tag = strings.TrimSpace(tag); stableTagRe.MatchString(tag) {
			return tag, nil
		}
	}
	return "", errors.New(
		"no stable Core release tag (core/vX.Y.Z); fetch the tags with " +
			"`git fetch --tags`",
	)
}

// extractRelease writes the schemas directory of the latest stable Core release into
// a new temporary directory. The caller must remove the returned Root.
func extractRelease(ctx context.Context, repoRoot string) (check.Release, error) {
	tag, err := latestStableTag(ctx, repoRoot)
	if err != nil {
		return check.Release{}, err
	}
	archive, err := exec.CommandContext(
		ctx, "git", "-C", repoRoot, "archive", "--format=tar", tag, "schemas",
	).Output()
	if err != nil {
		return check.Release{}, errors.Wrapf(err, "archive the schemas of %s", tag)
	}
	root, err := os.MkdirTemp("", "oracle-release-")
	if err != nil {
		return check.Release{}, err
	}
	if err := untar(archive, root); err != nil {
		return check.Release{}, errors.Join(err, os.RemoveAll(root))
	}
	return check.Release{Tag: tag, Root: root}, nil
}

func untar(archive []byte, root string) error {
	r := tar.NewReader(bytes.NewReader(archive))
	for {
		h, err := r.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		dest := filepath.Join(root, filepath.FromSlash(h.Name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return err
		}
	}
}
