// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFailed, onTestFinished, vi } from "vitest";

import { createRunningRack } from "@/platform/device/testutil";

const client = createTestClient();

describe("createRunningRack", () => {
  const cause = new Error("status write refused");
  let reported: unknown[] = [];

  it.fails("should fail the test that owns a rack whose report fails", async () => {
    onTestFailed(({ task }) => {
      reported = task.result?.errors?.map((e) => e.message) ?? [];
    });
    await createRunningRack(client, "sift");
    const set = vi.spyOn(client.statuses, "set").mockRejectedValueOnce(cause);
    onTestFinished(() => set.mockRestore());
    await waitFor(() => expect(set).toHaveBeenCalled());
  });

  it("should report the error of the failed report", () => {
    expect(reported).toEqual([cause.message]);
  });
});
