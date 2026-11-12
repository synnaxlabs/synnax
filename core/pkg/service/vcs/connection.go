// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package vcs

import (
	"os"
	"path/filepath"

	"github.com/synnaxlabs/x/errors"
)

const connectionFile = "connection.json"

// openReview is a pull request that holds some of the uncommitted changes.
type openReview struct {
	Number    int      `json:"number"`
	URL       string   `json:"url"`
	Resources []string `json:"resources"`
}

// connection is what the Core stores about version control apart from the repository.
// It holds the token, so the file is private to the owner of the Core.
type connection struct {
	Configuration string       `json:"configuration"`
	RemoteURL     string       `json:"remote_url"`
	Token         string       `json:"token"`
	Reviews       []openReview `json:"reviews"`
	// WaitingTags are the snapshot tags that wait for a push.
	WaitingTags []string `json:"waiting_tags"`
}

func (c connection) remote() remote { return remote{url: c.RemoteURL, token: c.Token} }

func readConnection(dir string) (connection, error) {
	b, err := os.ReadFile(filepath.Join(dir, connectionFile))
	if errors.Is(err, os.ErrNotExist) {
		return connection{}, nil
	}
	if err != nil {
		return connection{}, err
	}
	var c connection
	if err = decode(b, &c); err != nil {
		return connection{}, err
	}
	return c, nil
}

func writeConnection(dir string, c connection) error {
	b, err := encode(c)
	if err != nil {
		return err
	}
	p := filepath.Join(dir, connectionFile)
	if err = os.WriteFile(p, b, 0o600); err != nil {
		return err
	}
	return os.Chmod(p, 0o600)
}
