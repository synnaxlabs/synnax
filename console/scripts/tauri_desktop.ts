// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Runs a Tauri CLI command for Synnax Desktop: `node scripts/tauri_desktop.ts <command>
// [arguments...]`. It adds the Desktop config, and on Linux the config that renames the
// binary, since the Console .deb already installs /usr/bin/synnax-console.

import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";

const configs = ["--config", "src-tauri/tauri.desktop.conf.json"];
if (process.platform === "linux")
  configs.push("--config", "src-tauri/tauri.desktop.linux.conf.json");
const [command, ...args] = process.argv.slice(2);
// The CLI's own entry runs without a shell, so cmd.exe cannot strip the quotes of an
// argument on Windows.
const cli = createRequire(import.meta.url).resolve("@tauri-apps/cli/tauri.js");
const { status } = spawnSync(process.execPath, [cli, command, ...configs, ...args], {
  stdio: "inherit",
  env: { ...process.env, VITE_DESKTOP: "true" },
});
process.exit(status ?? 1);
