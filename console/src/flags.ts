// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Static build-time flags that hide unfinished work in the Console build. Vite replaces
 * `FLAGS_ON` and `import.meta.env.VITE_*` statically, so dark code tree-shakes out of
 * the Console build. `FLAGS_ON` turns every flag on in dev, Desktop, and spec builds.
 * Only the exact string "true" enables a flag. Each entry names its owner, its Linear
 * umbrella issue, and the release that removes it:
 *
 * `example: FLAGS_ON || import.meta.env.VITE_FLAG_EXAMPLE === "true", // Owner: Name.
 * Umbrella: SY-1234. Removed in 0.60.`
 */
export const FLAGS = {
  // Owner: Emiliano Bonilla. Umbrella: SY-5210. Removed in 0.61.
  ethercatStartupParameters:
    FLAGS_ON || import.meta.env.VITE_FLAG_ETHERCAT_STARTUP_PARAMETERS === "true",
} as const satisfies Record<string, boolean>;

export type Flag = keyof typeof FLAGS;
