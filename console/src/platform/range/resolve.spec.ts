// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeRange, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import { Session } from "@/session";
import { renderHookWithConsole } from "@/testutil";

const client = createTestClient();

const preloadedFor = (
  ranges: Session.Range.PersistedState[],
): { range: Session.Range.SliceState } => ({
  [Session.Range.SLICE_NAME]: { ...Session.Range.ZERO_SLICE_STATE, ranges },
});

const renderResolveMultiple = async (
  ranges: Session.Range.PersistedState[],
  keys?: string[],
) =>
  await renderHookWithConsole(() => Range.useResolveMultiple(keys), {
    client,
    preloadedState: preloadedFor(ranges),
  });

const renderResolve = async (ranges: Session.Range.PersistedState[], key: string) =>
  await renderHookWithConsole(() => Range.useResolve(key), {
    client,
    preloadedState: preloadedFor(ranges),
  });

describe("Range.useResolveMultiple", () => {
  // The session stores a Core range by key alone, so nothing renders until the Core
  // answers for it.
  it("should fill a Core range in from the Core", async () => {
    const created = await createTestRange(client);
    const { result } = await renderResolveMultiple(
      Session.Range.fromClient(created.payload),
      [created.key],
    );
    await waitFor(() =>
      expect(result.current).toEqual([
        {
          variant: "persisted",
          key: created.key,
          name: created.name,
          timeRange: created.timeRange,
        },
      ]),
    );
  });

  it("should keep the exact bounds of a Core range", async () => {
    // A float64 holds this start only to the nearest 256 ns.
    const start = new TimeStamp(TimeStamp.now().valueOf() + 1n);
    const end = start.add(TimeSpan.seconds(10));
    const created = await client.ranges.create({
      name: uniqueRangeName(),
      timeRange: new TimeRange(start, end),
    });
    const { result } = await renderResolveMultiple(
      Session.Range.fromClient(created.payload),
      [created.key],
    );
    await waitFor(() =>
      expect(result.current).toEqual([
        {
          variant: "persisted",
          key: created.key,
          name: created.name,
          timeRange: new TimeRange(start, end),
        },
      ]),
    );
  });

  it("should pass a rolling range through untouched", async () => {
    const { result } = await renderResolveMultiple([], [Session.Range.BUILT_IN[0].key]);
    expect(result.current).toEqual([Session.Range.BUILT_IN[0]]);
  });

  // A half-rendered row would show a Core range with no name until the synchronizer
  // caught up with the delete.
  it("should drop a range the Core no longer holds", async () => {
    const created = await createTestRange(client);
    await client.ranges.delete(created.key);
    const { result } = await renderResolveMultiple(
      Session.Range.fromClient(created.payload),
      [created.key],
    );
    await waitFor(() => expect(result.current).toEqual([]));
  });

  // A line plot can hold a Core range that is not in the viewer's session, for example
  // a plot another user made.
  it("should resolve a Core range the session does not hold", async () => {
    const created = await createTestRange(client);
    const { result } = await renderResolveMultiple([], [created.key]);
    await waitFor(() =>
      expect(result.current).toEqual([
        {
          variant: "persisted",
          key: created.key,
          name: created.name,
          timeRange: created.timeRange,
        },
      ]),
    );
  });

  it("should resolve the Core ranges it can when another is gone", async () => {
    const created = await createTestRange(client);
    const deleted = await createTestRange(client);
    await client.ranges.delete(deleted.key);
    const { result } = await renderResolveMultiple(
      Session.Range.fromClient(created.payload),
      [created.key, deleted.key],
    );
    await waitFor(() =>
      expect(result.current.map(({ key }) => key)).toEqual([created.key]),
    );
  });

  // A line plot's custom range has the key "custom", which is not a range key.
  it("should resolve Core ranges beside a key that is not a range key", async () => {
    const created = await createTestRange(client);
    const { result } = await renderResolveMultiple([], ["custom", created.key]);
    await waitFor(() =>
      expect(result.current.map(({ key }) => key)).toEqual([created.key]),
    );
  });

  it("should put a range the session does not hold after the session's", async () => {
    const outside = await createTestRange(client);
    const held = await createTestRange(client);
    const { result } = await renderResolveMultiple(
      Session.Range.fromClient(held.payload),
      [outside.key, held.key],
    );
    await waitFor(() =>
      expect(result.current.map(({ key }) => key)).toEqual([held.key, outside.key]),
    );
  });

  it("should keep the session's order", async () => {
    const first = await createTestRange(client);
    const second = await createTestRange(client);
    const { result } = await renderResolveMultiple(
      Session.Range.fromClient([second.payload, first.payload]),
      [first.key, second.key],
    );
    await waitFor(() =>
      expect(result.current.map(({ key }) => key)).toEqual([second.key, first.key]),
    );
  });
});

describe("Range.useResolve", () => {
  it("should resolve a Core range the session holds", async () => {
    const created = await createTestRange(client);
    const { result } = await renderResolve(
      Session.Range.fromClient(created.payload),
      created.key,
    );
    await waitFor(() => expect(result.current?.key).toBe(created.key));
    expect(result.current).toHaveProperty("name", created.name);
  });

  it("should resolve nothing for a key the session does not hold", async () => {
    const { result } = await renderResolve([], uuid.create());
    expect(result.current).toBeUndefined();
  });

  // The copy the session used to keep is what updateRemote existed to repair; reading
  // the Core instead is what makes a rename show through without one.
  it("should follow a rename made on the Core", async () => {
    const created = await createTestRange(client);
    const { result } = await renderResolve(
      Session.Range.fromClient(created.payload),
      created.key,
    );
    await waitFor(() => expect(result.current).toHaveProperty("name", created.name));
    const renamed = uniqueRangeName("renamed");
    await client.ranges.rename(created.key, renamed);
    await waitFor(() => expect(result.current).toHaveProperty("name", renamed));
  });
});
