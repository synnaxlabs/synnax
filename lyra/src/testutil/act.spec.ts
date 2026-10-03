// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { disableActEnvironment } from "@/testutil/act";

describe("disableActEnvironment", () => {
  it("should turn the act environment off", () => {
    disableActEnvironment();
    expect(globalThis.IS_REACT_ACT_ENVIRONMENT).toBe(false);
  });

  it("should keep the act environment off for the rest of the file", () => {
    expect(globalThis.IS_REACT_ACT_ENVIRONMENT).toBe(false);
  });
});
