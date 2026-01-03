// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, describe, expect, it } from "vitest";

import { instrumentation } from "@/dev";
import { type Instrumentation } from "@/instrumentation";
import { Tracer } from "@/trace";

describe("Trace", () => {
  let ins: Instrumentation;
  beforeAll(() => {
    ins = instrumentation();
  });
  describe("initialize", () => {
    it("should correctly initialize a tracer", () => {
      const t = new Tracer();
      expect(t).toBeDefined();
    });
  });
  describe("trace", () => {
    it("should start a span with the given key", () => {
      ins.T.prod("test", (span) => {
        expect(span.key).toEqual("test");
      });
    });
    it("should trace an async function correctly", async () => {
      const result = await ins.T.prod("async-test", async () => "test");
      expect(result).toEqual("test");
    });
  });
});
