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
	"bytes"
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/x/errors"
)

// stableTagRe matches a stable Core release tag and captures its X.Y train. A release
// candidate carries a suffix.
var stableTagRe = regexp.MustCompile(`^core/v([0-9]+\.[0-9]+)\.[0-9]+$`)

// runGit runs Git and returns its stdout. A failure carries Git's stderr.
func runGit(ctx context.Context, args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// stableTags returns the stable Core release tags that match pattern, newest first.
func stableTags(
	ctx context.Context, repoRoot, pattern string, args ...string,
) ([]string, error) {
	out, err := runGit(ctx, append(
		[]string{"-C", repoRoot, "tag", "--list", pattern, "--sort=-v:refname"},
		args...,
	)...)
	if err != nil {
		return nil, errors.Wrap(err, "list the Core release tags")
	}
	var tags []string
	for tag := range strings.Lines(string(out)) {
		if tag = strings.TrimSpace(tag); stableTagRe.MatchString(tag) {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

// shippedTag returns the release whose data HEAD must still decode: the newest stable
// tag on the train of the newest stable tag that HEAD descends from. On main, that is
// the newest release. On a release/X.Y branch, it is the newest X.Y patch.
func shippedTag(ctx context.Context, repoRoot string) (string, error) {
	reachable, err := stableTags(ctx, repoRoot, "core/v*", "--merged", "HEAD")
	if err != nil {
		return "", err
	}
	if len(reachable) == 0 {
		return "", errors.New(
			"HEAD descends from no stable Core release tag (core/vX.Y.Z); fetch " +
				"the tags with `git fetch --tags`, plus `--unshallow` in a shallow clone",
		)
	}
	train := stableTagRe.FindStringSubmatch(reachable[0])[1]
	onTrain, err := stableTags(ctx, repoRoot, "core/v"+train+".*")
	if err != nil {
		return "", err
	}
	return onTrain[0], nil
}

// extractRelease writes the schemas directory of the shipped release into a new
// temporary directory. The caller must remove the returned Root.
func extractRelease(ctx context.Context, repoRoot string) (check.Release, error) {
	tag, err := shippedTag(ctx, repoRoot)
	if err != nil {
		return check.Release{}, err
	}
	archive, err := runGit(
		ctx, "-C", repoRoot, "archive", "--format=tar", tag, "schemas",
	)
	if err != nil {
		return check.Release{}, errors.Wrapf(err, "archive the schemas of %s", tag)
	}
	root, err := os.MkdirTemp("", "oracle-release-")
	if err != nil {
		return check.Release{}, err
	}
	untar := exec.CommandContext(ctx, "tar", "-x", "-f", "-", "-C", root)
	untar.Stdin = bytes.NewReader(archive)
	if out, err := untar.CombinedOutput(); err != nil {
		return check.Release{}, errors.Join(
			errors.Wrapf(err, "extract the schemas of %s: %s", tag, out),
			os.RemoveAll(root),
		)
	}
	return check.Release{Tag: tag, Root: root}, nil
}
