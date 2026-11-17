// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeRange, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Session } from "@/session";
import { createConsoleWrapper, selectRange, uniqueName } from "@/testutil";

const client = createTestClient();

const createRange = async (): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name: uniqueName("range"),
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(10))),
  });
};

const preloadedFor = (keys: string[]) => ({
  [Session.Range.SLICE_NAME]: Session.Range.sliceStateZ.parse({
    ranges: keys.map((key) => ({ variant: "persisted", key })),
  }),
});

describe("Range.SYNCHRONIZERS", () => {
  it("should remove a favorited range from the slice when it is deleted remotely", async () => {
    const range = await createRange();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: preloadedFor([range.key]),
    });
    renderHook(() => Session.Synchronizer.use(Session.Range.SYNCHRONIZERS), {
      wrapper,
    });
    expect(selectRange(store.getState(), range.key)).toBeDefined();
    await client.ranges.delete(range.key);
    await waitFor(() => {
      expect(selectRange(store.getState(), range.key)).toBeUndefined();
    });
  });

  it("should remove a favorited range the Core no longer holds on connect", async () => {
    const range = await createRange();
    const gone = uuid.create();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: preloadedFor([range.key, gone]),
    });
    renderHook(() => Session.Synchronizer.use(Session.Range.SYNCHRONIZERS), {
      wrapper,
    });
    await waitFor(() => {
      expect(selectRange(store.getState(), gone)).toBeUndefined();
    });
    expect(selectRange(store.getState(), range.key)).toBeDefined();
  });
});
