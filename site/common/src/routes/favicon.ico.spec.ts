// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { GET } from "./favicon.ico";

describe("favicon.ico", () => {
  it("should respond with an icon image", async () => {
    const response = await GET();
    expect(response.headers.get("Content-Type")).toBe("image/x-icon");
    expect((await response.arrayBuffer()).byteLength).toBeGreaterThan(0);
  });
});
