// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// HACK: the recursive directory walks sweep up files the bundle does not own — Finder's
// .DS_Store, AppleDouble ._*.json copies, stray notes — and one JSON.parse failure
// aborts the whole import. Filter to plain .json files before parsing until server-side
// project import owns decoding.

/** Reports whether the file at path is a plain .json file worth parsing. */
export const isParsableFile = (path: string): boolean => {
  const base = path.slice(path.lastIndexOf("/") + 1);
  return base.endsWith(".json") && !base.startsWith(".");
};
