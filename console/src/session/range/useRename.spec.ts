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
import { TimeRange, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
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

const localState = (): Session.Range.StaticState => {
  const start = TimeStamp.now();
  return {
    key: uuid.create(),
    name: uniqueName("local"),
    persisted: false,
    variant: "static",
    timeRange: { start: Number(start), end: Number(start.add(TimeSpan.seconds(1))) },
  };
};

const preloadedFor = (ranges: Session.Range.State[]) => ({
  [Session.Range.SLICE_NAME]: { version: 0 as const, ranges },
});

const renderRename = async (ranges: Session.Range.State[]) => {
  const { wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: preloadedFor(ranges),
  });
  const { result } = renderHook(Session.Range.useRename, { wrapper });
  return { result, store };
};

describe("Range.useRename", () => {
  it("should rename a persisted range on the Core and in the slice", async () => {
    const range = await createRange();
    const [state] = Session.Range.fromClient(range);
    const { result, store } = await renderRename([state]);
    const renamed = uniqueName("renamed");
    await act(async () => {
      await result.current.updateAsync({ key: range.key, name: renamed });
    });
    await waitFor(() =>
      expect(Session.Range.selectState(store.getState(), range.key)?.name).toBe(
        renamed,
      ),
    );
    expect((await client.ranges.retrieve(range.key)).name).toBe(renamed);
  });

  it("should rename a local range in the slice without reaching the Core", async () => {
    const local = localState();
    const { result, store } = await renderRename([local]);
    const renamed = uniqueName("renamed");
    await act(async () => {
      await result.current.updateAsync({ key: local.key, name: renamed });
    });
    await waitFor(() =>
      expect(Session.Range.selectState(store.getState(), local.key)?.name).toBe(
        renamed,
      ),
    );
    await expect(client.ranges.retrieve(local.key)).rejects.toThrow(NotFoundError);
  });

  it("should roll the slice back when the Core rejects the rename", async () => {
    const range = await createRange();
    const [state] = Session.Range.fromClient(range);
    const { result, store } = await renderRename([state]);
    await client.ranges.delete(range.key);
    await act(async () => {
      await result.current.updateAsync({ key: range.key, name: uniqueName("renamed") });
    });
    await waitFor(() =>
      expect(Session.Range.selectState(store.getState(), range.key)?.name).toBe(
        state.name,
      ),
    );
  });

  it("should leave the slice alone for a range it does not hold", async () => {
    const range = await createRange();
    const { result, store } = await renderRename([]);
    const renamed = uniqueName("renamed");
    await act(async () => {
      await result.current.updateAsync({ key: range.key, name: renamed });
    });
    await waitFor(async () =>
      expect((await client.ranges.retrieve(range.key)).name).toBe(renamed),
    );
    expect(Session.Range.selectState(store.getState(), range.key)).toBeUndefined();
  });
});
