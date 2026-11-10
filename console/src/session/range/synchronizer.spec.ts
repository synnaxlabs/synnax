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
import { staticWindow, TimeRange, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
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

const preloadedFor = (range: ranger.Range) => ({
  [Session.Range.SLICE_NAME]: {
    version: 0 as const,
    ranges: Session.Range.fromClient(range),
  },
});

describe("Range.SYNCHRONIZERS", () => {
  it("should remove a favorited range from the slice when it is deleted remotely", async () => {
    const range = await createRange();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: preloadedFor(range),
    });
    renderHook(() => Session.Synchronizer.use(Session.Range.SYNCHRONIZERS), {
      wrapper,
    });
    expect(Session.Range.selectState(store.getState(), range.key)).toBeDefined();
    await client.ranges.delete(range.key);
    await waitFor(() => {
      expect(Session.Range.selectState(store.getState(), range.key)).toBeUndefined();
    });
  });

  it("should show a session range's window on a plot that stored its key", async () => {
    const local: Session.Range.StaticState = {
      variant: "static",
      key: uuid.create(),
      name: uniqueName("local"),
      timeRange: { start: 1, end: 2 },
    };
    const project = await client.projects.create({ name: uniqueName("project") });
    const plot = await client.lineplots.create({
      project: project.key,
      name: uniqueName("plot"),
      ranges: [{ variant: "key", key: local.key }],
    });
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: {
        [Session.Range.SLICE_NAME]: { version: 0 as const, ranges: [local] },
      },
    });
    renderHook(() => Session.Synchronizer.use(Session.Range.SYNCHRONIZERS), {
      wrapper,
    });
    const reader = createTestClient();
    await waitFor(async () => {
      const { ranges } = await reader.lineplots.retrieve(plot.key);
      expect(ranges).toEqual([
        {
          variant: "window",
          window: staticWindow(new TimeRange(local.timeRange)),
        },
      ]);
    });
  });
});
