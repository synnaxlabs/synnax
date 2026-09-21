// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Turns a `VITE_FLAG_*` value into a flag. Vite replaces `import.meta.env.VITE_*`
 * statically, so dark code tree-shakes out of production. Dev builds turn every flag
 * on.
 */
export const flag = (value: string | undefined): boolean => IS_DEV || value === "true";

/**
 * Static build-time flags that hide unfinished work in production. Each entry names
 * its owner and the release that removes it:
 * `example: flag(import.meta.env.VITE_FLAG_EXAMPLE), // Owner: Name. Removed in 0.60.`
 */
export const FLAGS = {} as const satisfies Record<string, boolean>;

export type Flag = keyof typeof FLAGS;
