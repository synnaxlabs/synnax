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
 * the build constants statically, so dark code tree-shakes out of the Console build. A
 * flag is on only where its expression says. Use `IS_DEV` for dev builds and
 * `import.meta.env.VITE_FLAGS_ENABLED === "true"` for the integration build and specs.
 * Release builds leave a flag off, except that a Desktop release turns on a flag whose
 * expression is true when `DESKTOP` is. Each entry names its owner, its Linear
 * umbrella issue, and the release that removes it.
 */
export const FLAGS = {
  // Owner: Emiliano Bonilla. Umbrella: SY-3591. Removed in the release that ships the
  // Sift integration.
  sift: IS_DEV || import.meta.env.VITE_FLAGS_ENABLED === "true",
} as const satisfies Record<string, boolean>;

export type Flag = keyof typeof FLAGS;
