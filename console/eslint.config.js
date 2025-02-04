// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { includeIgnoreFile } from "@eslint/compat";
import baseConfig from "eslint-config-synnaxlabs";
import path from "path";
import { fileURLToPath } from "url";

const filename = fileURLToPath(import.meta.url);
const dirname = path.dirname(filename);
const gitignorePaths = [".gitignore", "src-tauri/.gitignore"];

const ignoredFiles = gitignorePaths.map((p) => {
  const gitignorePath = path.join(dirname, p);
  return includeIgnoreFile(gitignorePath);
});

export default [...ignoredFiles, ...baseConfig, { ignores: [".vite/"] }];
