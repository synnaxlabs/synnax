// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Static build-time flags that hide unfinished work in production. Vite replaces
 * `import.meta.env.VITE_*` statically, so dark code tree-shakes out of production, and
 * dev builds turn every flag on. Only the exact string "true" enables a flag. Each
 * entry names its owner and the release that removes it:
 * `example: IS_DEV || import.meta.env.VITE_FLAG_EXAMPLE === "true", // Owner: Name.
 * Removed in 0.60.`
 */
export const FLAGS = {} as const satisfies Record<string, boolean>;

export type Flag = keyof typeof FLAGS;
