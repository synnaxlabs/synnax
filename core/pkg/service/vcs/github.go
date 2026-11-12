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
	"bytes"
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

const maxSlugLength = 40

// slug turns a commit message into a branch name: its first line in lowercase, with
// each run of other characters as one dash.
func slug(message string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(message), "\n")
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(line) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
		if b.Len() >= maxSlugLength {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}

// repoPath returns the owner and the name of the repository at a GitHub remote URL.
func repoPath(remoteURL string) (string, string, error) {
	p := remoteURL
	if u, err := url.Parse(remoteURL); err == nil && u.Scheme != "" {
		p = u.Path
	} else if _, after, ok := strings.Cut(remoteURL, ":"); ok {
		p = after
	}
	segments := strings.Split(strings.Trim(strings.TrimSuffix(p, ".git"), "/"), "/")
	if len(segments) < 2 {
		return "", "", errors.Wrapf(
			validate.ErrValidation,
			"remote %q does not name a GitHub repository",
			remoteURL,
		)
	}
	return segments[len(segments)-2], segments[len(segments)-1], nil
}

type pullRequestBody struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body"`
}

type pullRequestResponse struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	Message string `json:"message"`
}

// openPullRequest pushes the commit h to a new review branch and opens a pull request
// from it into the tracked configuration.
func (s *Service) openPullRequest(
	ctx context.Context,
	message string,
	h plumbing.Hash,
) (openReview, error) {
	branch := reviewPrefix + strings.Trim(
		slug(message)+"-"+h.String()[:shortHashSize],
		"-",
	)
	if err := s.repo.setRef(branchRef(branch), h); err != nil {
		return openReview{}, err
	}
	if err := s.repo.push(ctx, s.conn.remote(), branch); err != nil {
		return openReview{}, errors.Wrap(err, "push the review branch")
	}
	owner, name, err := repoPath(s.conn.RemoteURL)
	if err != nil {
		return openReview{}, err
	}
	title, rest, _ := strings.Cut(strings.TrimSpace(message), "\n")
	reqBody, err := json.Marshal(pullRequestBody{
		Title: title,
		Head:  branch,
		Base:  s.conn.Configuration,
		Body:  strings.TrimSpace(rest),
	})
	if err != nil {
		return openReview{}, err
	}
	endpoint := strings.TrimSuffix(s.cfg.GitHubAPI, "/") +
		"/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/pulls"
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return openReview{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if s.conn.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.conn.Token)
	}
	res, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return openReview{}, errors.Wrap(err, "open the pull request")
	}
	resBody, err := io.ReadAll(res.Body)
	if err = errors.Combine(err, res.Body.Close()); err != nil {
		return openReview{}, err
	}
	var pr pullRequestResponse
	if err = json.Unmarshal(
		resBody,
		&pr,
		json.RejectUnknownMembers(false),
	); err != nil {
		return openReview{}, errors.Wrapf(err, "GitHub returned %s", res.Status)
	}
	if res.StatusCode != http.StatusCreated {
		return openReview{}, errors.Newf(
			"GitHub refused the pull request: %s %s",
			res.Status,
			pr.Message,
		)
	}
	return openReview{Number: pr.Number, URL: pr.HTMLURL}, nil
}
