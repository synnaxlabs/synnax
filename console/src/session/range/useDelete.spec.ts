// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const createRange = async (): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name: uniqueName("range"),
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(10))),
  });
};

// The Core rejects a key that is not a UUID, which is how a delete is made to fail.
const REJECTED_KEY = "not-a-uuid";

const rejectedState = (): Session.Range.StaticState => {
  const start = TimeStamp.now();
  return {
    key: REJECTED_KEY,
    name: uniqueName("rejected"),
    persisted: true,
    variant: "static",
    timeRange: { start: Number(start), end: Number(start.add(TimeSpan.seconds(1))) },
  };
};

const renderDelete = async (ranges: Session.Range.State[], selected?: string) => {
  const { wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: {
      [Session.Range.SLICE_NAME]: { version: 0 as const, ranges, selected },
    },
  });
  const { result } = renderHook(Session.Range.useDelete, { wrapper });
  return { result, store };
};

describe("Range.useDelete", () => {
  it("should delete the range on the Core and drop it from the slice", async () => {
    const range = await createRange();
    const [state] = Session.Range.fromClient(range);
    const { result, store } = await renderDelete([state], range.key);
    await act(async () => {
      await result.current.updateAsync(range.key);
    });
    await waitFor(() =>
      expect(Session.Range.selectState(store.getState(), range.key)).toBeUndefined(),
    );
    expect(Session.Range.selectSelectedKey(store.getState())).toBeUndefined();
    await expect(client.ranges.retrieve(range.key)).rejects.toThrow(NotFoundError);
  });

  it("should roll the range back to its place when the Core rejects the delete", async () => {
    const range = await createRange();
    const [neighbor] = Session.Range.fromClient(range);
    const rejected = rejectedState();
    const { result, store } = await renderDelete([rejected, neighbor], REJECTED_KEY);
    await act(async () => {
      await result.current.updateAsync(REJECTED_KEY);
    });
    await waitFor(() =>
      expect(Session.Range.selectKeys(store.getState())).toEqual([
        REJECTED_KEY,
        neighbor.key,
      ]),
    );
    expect(Session.Range.selectSelectedKey(store.getState())).toEqual(REJECTED_KEY);
  });

  it("should leave the slice alone for a range it does not hold", async () => {
    const range = await createRange();
    const { result, store } = await renderDelete([]);
    await act(async () => {
      await result.current.updateAsync(range.key);
    });
    await waitFor(
      async () =>
        await expect(client.ranges.retrieve(range.key)).rejects.toThrow(NotFoundError),
    );
    expect(Session.Range.selectKeys(store.getState())).toEqual([]);
  });
});
