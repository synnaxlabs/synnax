// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

// The x setup file calls failOnConsoleOutput, so these specs run under the check.
describe("failOnConsoleOutput", () => {
  it.fails("should fail a test that writes to console.error", () => {
    console.error("unexpected error");
  });

  it.fails("should fail a test that writes to console.warn", () => {
    console.warn("unexpected warning");
  });

  it.fails("should fail a warning that only names a Redux Toolkit check", () => {
    console.warn("ImmutableStateInvariantMiddleware found a mutation");
  });

  it("should pass a test whose Redux Toolkit checks ran slowly", () => {
    console.warn(
      "ImmutableStateInvariantMiddleware took 36ms, which is more than the warning threshold of 32ms.",
    );
    console.warn(
      "SerializableStateInvariantMiddleware took 36ms, which is more than the warning threshold of 32ms.",
    );
  });

  it("should start each test with no recorded output", () => {});

  it("should pass a test that mocks the method it expects", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    console.warn("expected warning");
    expect(warn).toHaveBeenCalledWith("expected warning");
  });
});
