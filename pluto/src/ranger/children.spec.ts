// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, failRequests } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Ranger } from "@/ranger";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const createRange = async (name: string, start: TimeStamp, parent?: string) =>
  await client.ranges.create({
    name,
    timeRange: start.spanRange(TimeSpan.seconds(1)),
    parent,
  });

describe("useChildren", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it("should read the children of each parent in the order they ran", async () => {
    const start = TimeStamp.now();
    const parent = await createRange("parent", start);
    const late = await createRange("late", start.add(TimeSpan.seconds(2)), parent.key);
    const early = await createRange("early", start, parent.key);
    const { result } = renderHook(() => Ranger.useChildren({ parents: [parent.key] }), {
      wrapper,
    });
    await waitFor(() =>
      expect(result.current.children.get(parent.key)).toEqual({
        keys: [early.key, late.key],
        complete: true,
      }),
    );
  });

  it("should show a child that a range gains while it reads", async () => {
    const start = TimeStamp.now();
    const parent = await createRange("parent", start);
    const { result } = renderHook(() => Ranger.useChildren({ parents: [parent.key] }), {
      wrapper,
    });
    await waitFor(() =>
      expect(result.current.children.get(parent.key)?.keys).toEqual([]),
    );
    const child = await createRange("child", start, parent.key);
    await waitFor(() =>
      expect(result.current.children.get(parent.key)?.keys).toEqual([child.key]),
    );
  });

  it("should read the next page of a parent with more children", async () => {
    const start = TimeStamp.now();
    const parent = await createRange("parent", start);
    await client.ranges.create(
      Array.from({ length: 51 }, (_, i) => ({
        name: `child ${i}`,
        timeRange: start.add(TimeSpan.seconds(i)).spanRange(TimeSpan.seconds(1)),
        parent: parent.key,
      })),
    );
    const { result } = renderHook(() => Ranger.useChildren({ parents: [parent.key] }), {
      wrapper,
    });
    await waitFor(() =>
      expect(result.current.children.get(parent.key)).toMatchObject({
        complete: false,
      }),
    );
    expect(result.current.children.get(parent.key)?.keys).toHaveLength(50);
    act(() => result.current.more(parent.key));
    await waitFor(() =>
      expect(result.current.children.get(parent.key)).toMatchObject({
        complete: true,
      }),
    );
    expect(result.current.children.get(parent.key)?.keys).toHaveLength(51);
  });

  it("should add an error status when a read fails", async () => {
    const parent = await createRange("parent", TimeStamp.now());
    const reader = createTestClient();
    const readerWrapper = await createAsyncSynnaxWrapper({ client: reader });
    failRequests(reader, "/range/retrieve", new Error("Core unreachable"));
    const { result } = renderHook(
      () => ({
        ...Ranger.useChildren({ parents: [parent.key] }),
        statuses: Status.useNotifications().statuses,
      }),
      { wrapper: readerWrapper },
    );
    await waitFor(() =>
      expect(result.current.statuses).toContainEqual(
        expect.objectContaining({
          variant: "error",
          message: "Failed to retrieve child ranges",
          description: "Core unreachable",
        }),
      ),
    );
    expect(result.current.children.has(parent.key)).toBe(false);
  });
});
