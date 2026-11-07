// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package console

import (
	"encoding/json"
	"io/fs"
	"net/url"
	"path"

	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

const updatePath = "/console/update/"

// manifestName is the file name of the Tauri updater manifest of a Console release.
const manifestName = "latest.json"

// manifest is a Tauri updater manifest in the static format.
type manifest struct {
	Version   string              `json:"version"`
	Notes     string              `json:"notes,omitempty"`
	PubDate   string              `json:"pub_date,omitempty"`
	Platforms map[string]platform `json:"platforms"`
}

// platform is the update for one updater target, such as windows-x86_64-msi.
type platform struct {
	Signature string `json:"signature"`
	URL       string `json:"url"`
}

// update is the desktop Console update that the Core embeds. It is empty when the Core
// embeds no file that the manifest names.
type update struct {
	fs fs.FS
	// manifest holds the file name in place of each platform URL.
	manifest manifest
	// files holds the name of each file that manifest names.
	files set.Set[string]
}

// openUpdate reads the manifest in fsys and keeps the platforms whose file fsys holds.
func openUpdate(fsys fs.FS) (update, error) {
	data, err := fs.ReadFile(fsys, manifestName)
	if errors.Is(err, fs.ErrNotExist) {
		return update{}, nil
	}
	if err != nil {
		return update{}, err
	}
	var release manifest
	if err = json.Unmarshal(data, &release); err != nil {
		return update{}, errors.Wrap(err, "failed to parse the Console update manifest")
	}
	u := update{fs: fsys, manifest: release, files: set.New[string]()}
	u.manifest.Platforms = make(map[string]platform, len(release.Platforms))
	for target, p := range release.Platforms {
		fileURL, err := url.Parse(p.URL)
		if err != nil {
			return update{}, errors.Wrapf(err, "failed to parse the URL of %s", target)
		}
		name := path.Base(fileURL.Path)
		if _, err = fs.Stat(fsys, name); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return update{}, err
		}
		p.URL = name
		u.manifest.Platforms[target] = p
		u.files.Add(name)
	}
	return u, nil
}

// sendUpdateManifest sends the release manifest with each platform URL pointing at the
// host that the request names. An X-Forwarded-Proto header sets the scheme, so a Core
// behind a TLS proxy names the scheme of the proxy.
func (c *Console) sendUpdateManifest(ctx fiber.Ctx) error {
	if len(c.update.files) == 0 {
		return fiber.ErrNotFound
	}
	scheme := c.scheme
	if proto := ctx.Get(
		fiber.HeaderXForwardedProto,
	); proto == "http" ||
		proto == "https" {
		scheme = proto
	}
	base := scheme + "://" + ctx.Host() + updatePath
	served := c.update.manifest
	served.Platforms = make(map[string]platform, len(c.update.manifest.Platforms))
	for target, p := range c.update.manifest.Platforms {
		p.URL = base + p.URL
		served.Platforms[target] = p
	}
	ctx.Set(fiber.HeaderCacheControl, noStoreCacheControl)
	return ctx.JSON(served)
}

func (c *Console) sendUpdateFile(ctx fiber.Ctx) error {
	name := ctx.Params("name")
	if !c.update.files.Contains(name) {
		return fiber.ErrNotFound
	}
	if err := ctx.SendFile(
		name,
		fiber.SendFile{FS: c.update.fs, CacheDuration: -1},
	); err != nil {
		return err
	}
	ctx.Set(fiber.HeaderCacheControl, noStoreCacheControl)
	return nil
}
