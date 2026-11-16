// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  type MockInstance,
  vi,
} from "vitest";

import { Instrumentation, Logger } from "@/index";

describe("Logger", () => {
  describe("NOOP", () => {
    let error: MockInstance;
    beforeEach(() => {
      error = vi.spyOn(console, "error").mockImplementation(() => {});
    });
    afterEach(() => error.mockRestore());

    it("should not write to the console", () => {
      Logger.NOOP.error("boom");
      expect(error).not.toHaveBeenCalled();
    });

    it("should not write to the console from an instrumentation", () => {
      new Instrumentation({ key: "test" }).L.error("boom");
      expect(error).not.toHaveBeenCalled();
    });
  });
});
