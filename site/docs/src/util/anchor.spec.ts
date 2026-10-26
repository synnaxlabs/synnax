// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { toAnchor } from "@/util/anchor";

describe("toAnchor", () => {
  it("should join lowercase words with dashes", () => {
    expect(toAnchor("Vibration and sound")).toBe("vibration-and-sound");
  });

  it("should replace each run of other characters with one dash", () => {
    expect(toAnchor("Channel `key` (index)")).toBe("channel-key-index");
  });

  it("should drop a dash at either end", () => {
    expect(toAnchor("--Port?")).toBe("port");
  });
});
