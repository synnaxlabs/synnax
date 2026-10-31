// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { issueStatuses } from "@/platform/task/issues";

const issue = (
  path: PropertyKey[],
  message: string,
  variant?: "warning",
): z.core.$ZodIssue => ({
  code: "custom",
  path,
  message,
  input: null,
  params: variant == null ? undefined : { variant },
});

const values = { config: { channels: [{ key: "a" }, { key: "b" }] } };

describe("issueStatuses", () => {
  it("should key each status by the path of the field, with items by their key", () => {
    expect(issueStatuses([issue(["channels", 1, "port"], "taken")], values)).toEqual(
      new Map([
        [
          "config.channels.b.port",
          { key: "config.channels.b.port", variant: "error", message: "taken" },
        ],
      ]),
    );
  });

  it("should keep the first issue on a path", () => {
    const statuses = issueStatuses(
      [issue(["sampleRate"], "first"), issue(["sampleRate"], "second")],
      values,
    );
    expect(statuses.get("config.sampleRate")?.message).toBe("first");
  });

  it("should let an error outrank a warning on the same path", () => {
    const statuses = issueStatuses(
      [
        issue(["sampleRate"], "slow", "warning"),
        issue(["sampleRate"], "too fast"),
        issue(["sampleRate"], "late", "warning"),
      ],
      values,
    );
    expect(statuses.get("config.sampleRate")).toMatchObject({
      variant: "error",
      message: "too fast",
    });
  });
});
