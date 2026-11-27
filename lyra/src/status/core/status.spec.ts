// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Status } from "@/status";

describe("fromException", () => {
  it("should put the description of the heading before the error", () => {
    const stat = Status.fromException(new Error("Disk is full"), {
      message: "Failed to save",
      name: "Pump",
      description: "Free some space.",
    });
    expect(stat).toMatchObject({
      variant: "error",
      message: "Failed to save",
      name: "Pump",
      description: "Free some space. Disk is full",
    });
  });
});
