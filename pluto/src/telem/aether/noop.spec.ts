// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { NoopFactory, noopLogSourceSpec } from "@/telem/aether/noop";
import { type LogSource } from "@/telem/aether/telem";

describe("NoopLogSource", () => {
  it("should return an empty array from value()", () => {
    const factory = new NoopFactory();
    const source = factory.create(noopLogSourceSpec) as LogSource;
    expect(source.value()).toEqual([]);
  });

  it("should have evictedCount of 0", () => {
    const factory = new NoopFactory();
    const source = factory.create(noopLogSourceSpec) as LogSource;
    expect(source.evictedCount).toBe(0);
  });
});
