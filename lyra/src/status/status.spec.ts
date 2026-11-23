// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { fromException } from "@/status/status";

describe("fromException", () => {
  it("should chain the messages of an error and its causes with a colon", () => {
    const exc = new Error("failed to save", { cause: new Error("disk is full") });
    expect(fromException(exc, "Failed").description).toBe(
      "failed to save: disk is full",
    );
  });

  it("should lead a cause with a space after a message that ends a sentence", () => {
    const exc = new Error("Failed to fetch built-in channels.", {
      cause: new Error("Cannot reach Core at localhost:9090"),
    });
    expect(fromException(exc, "Failed").description).toBe(
      "Failed to fetch built-in channels. Cannot reach Core at localhost:9090",
    );
  });
});
