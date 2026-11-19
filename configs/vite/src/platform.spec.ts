// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { platformExcludes } from "./platform.js";

describe("platformExcludes", () => {
  it.each([
    ["linux", ["darwin", "win32"]],
    ["darwin", ["linux", "win32"]],
    ["win32", ["darwin", "linux", "posix"]],
  ])(
    "should exclude the test files of the other platforms on %s",
    (platform, others) => {
      expect(platformExcludes(platform)).toEqual(
        others.map((p) => `**/*.${p}.{spec,test}.{ts,tsx}`),
      );
    },
  );

  it("should take the platform of the process by default", () => {
    expect(platformExcludes()).toEqual(platformExcludes(process.platform));
  });
});
