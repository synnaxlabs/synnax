// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

import { discoverModules, moduleExports } from "@synnaxlabs/vite-plugin";

const root = path.resolve(import.meta.dirname, "..");
const manifestPath = path.join(root, "package.json");
const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
manifest.exports = moduleExports(discoverModules(root));
writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
