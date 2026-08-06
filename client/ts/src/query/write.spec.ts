// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { query } from "@/query";

describe("optimistic", () => {
  it("returns the commit result and keeps the optimistic state", async () => {
    const rollback = vi.fn();
    const result = await query.optimistic({
      rollbacks: [rollback],
      commit: async () => "committed",
    });
    expect(result).toEqual("committed");
    expect(rollback).not.toHaveBeenCalled();
  });

  it("runs the rollbacks in reverse order and rethrows when commit fails", async () => {
    const order: string[] = [];
    await expect(
      query.optimistic({
        rollbacks: [() => order.push("first"), () => order.push("second")],
        commit: async () => {
          throw new Error("send failed");
        },
      }),
    ).rejects.toThrow("send failed");
    expect(order).toEqual(["second", "first"]);
  });

  it("notifies onOptimistic after capturing rollbacks and before commit", async () => {
    const order: string[] = [];
    await query.optimistic({
      rollbacks: [() => order.push("rollback")],
      onOptimistic: () => {
        order.push("optimistic");
      },
      commit: async () => order.push("commit"),
    });
    expect(order).toEqual(["optimistic", "commit"]);
  });

  it("rolls back and skips commit when onOptimistic fails", async () => {
    const rollback = vi.fn();
    const commit = vi.fn(async () => "committed");
    await expect(
      query.optimistic({
        rollbacks: [rollback],
        onOptimistic: async () => {
          throw new Error("tab failed to open");
        },
        commit,
      }),
    ).rejects.toThrow("tab failed to open");
    expect(rollback).toHaveBeenCalledTimes(1);
    expect(commit).not.toHaveBeenCalled();
  });
});
