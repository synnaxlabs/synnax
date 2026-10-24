// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { status } from "@/status";
import { task } from "@/task";

const STATUS = status.create({
  variant: "error",
  message: "Failed to connect to server",
  description: "Connection refused",
});

describe("StatusError", () => {
  it("should return the status message and description from toStatus", () => {
    expect(new task.StatusError(STATUS).toStatus()).toEqual({
      message: "Failed to connect to server",
      description: "Connection refused",
    });
  });

  it("should keep the status message in a status built from it", () => {
    const s = status.fromException(new task.StatusError(STATUS), "Failed to connect");
    expect(s.message).toBe("Failed to connect: Failed to connect to server");
  });

  it("should keep the status description in a status built from it", () => {
    const s = status.fromException(new task.StatusError(STATUS), "Failed to connect");
    expect(s.description).toBe("Connection refused");
  });
});
