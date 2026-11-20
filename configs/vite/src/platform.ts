// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const PLATFORMS = ["darwin", "linux", "win32"];

/**
 * Returns the globs of the test files that do not run on platform. A test file named
 * for a platform, as in `a.darwin.spec.ts`, runs only there, and one named for `posix`
 * runs everywhere but `win32`. Add the globs to `test.exclude`, after
 * `configDefaults.exclude` of `vitest/config`, which an `exclude` replaces.
 *
 * @param platform - The platform the tests run on, as `process.platform` names it.
 */
export const platformExcludes = (platform: string = process.platform): string[] => {
  const excluded = PLATFORMS.filter((p) => p !== platform);
  if (platform === "win32") excluded.push("posix");
  return excluded.map((p) => `**/*.${p}.{spec,test}.{ts,tsx}`);
};
