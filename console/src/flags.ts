// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Static build-time flags that hide new work from the Console build. A flag is on in
 * dev and Desktop builds, where a feature ships first. Each flag is a top-level
 * constant, because the bundler tree-shakes code behind an inlined constant but not
 * behind an object property. Each flag names its owner, its Linear umbrella issue, and
 * the release that removes it:
 *
 * ```ts
 * // Owner: Name. Umbrella: SY-1234. Removed in 0.60.
 * export const EXAMPLE = IS_DEV || DESKTOP;
 * ```
 */

// Owner: Emiliano Bonilla. Umbrella: SY-1901. Removed in 0.60.
export const PROBLEMS_VIEW = IS_DEV || DESKTOP;
