// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { testCoreAddress } from "@/testutil";

describe("testCoreAddress", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("should return the provided Core over the environment", () => {
    vi.stubEnv("SYNNAX_TEST_HOST", "environment");
    vi.stubEnv("SYNNAX_TEST_PORT", "2");
    expect(testCoreAddress()).toEqual({ host: "provided", port: 1 });
  });
});
