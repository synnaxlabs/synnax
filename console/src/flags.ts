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
 * flag is on in dev and Desktop builds, and in a build that sets `VITE_FLAGS_ENABLED`
 * to the exact string "true", such as the integration build. A flag for work that does
 * not run yet leaves out Desktop builds. Each entry names its owner, its Linear
 * umbrella issue, and the release that removes it.
 */
const INTEGRATION = import.meta.env.VITE_FLAGS_ENABLED === "true";

export const FLAGS = {
  // Owner: Nico Alba. Umbrella: SY-4845. Removed in 0.62, when Arc modules ship. Module
  // code does not run yet, so Desktop builds leave it out.
  arcModules: IS_DEV || INTEGRATION,
} as const satisfies Record<string, boolean>;

export type Flag = keyof typeof FLAGS;
