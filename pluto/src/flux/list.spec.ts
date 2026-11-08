// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, type ranger } from "@synnaxlabs/client";
import {
  createTestClient,
  holdResponses,
  spyOnSend,
} from "@synnaxlabs/client/testutil";
import { color, sleep, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });
const ranges = (c: typeof client) => c.ranges;

// The Core tokenizes a search term on punctuation, so a term must be one word.
const randomWord = (): string => `rng${Math.random().toString(36).slice(2)}`;

// The Core indexes a new name for search a moment after the create returns.
const waitForIndex = async (name: string): Promise<void> => {
  await expect
    .poll(async () => (await client.ranges.retrieve({ searchTerm: name })).length)
    .toBeGreaterThan(0);
};

describe("useList", () => {
  let lbl: label.Label;
  const create = async (name: string): Promise<ranger.Range> =>
    await client.ranges.create({
      name,
      timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      labels: [lbl.key],
    });

  beforeEach(async () => {
    lbl = await client.labels.create({
      name: uuid.create(),
      color: color.construct("#000000"),
    });
  });

  const byName = () =>
    ({ where: { labels: [lbl.key] }, order: { field: "name" } }) as const;

  it("should list the matching records in the declared order", async () => {
    const b = await create("b");
    const a = await create("a");
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      {
        wrapper,
      },
    );
    expect(result.current.variant).toEqual("loading");
    expect(result.current.answered).toBe(false);
    act(() => result.current.more());
    await waitFor(() => {
      expect(result.current.variant).toEqual("success");
      expect(result.current.data).toEqual([a.key, b.key]);
    });
    expect(result.current.answered).toBe(true);
    expect(result.current.getItem(a.key)?.name).toEqual("a");
  });

  it("should place a created record in order", async () => {
    const a = await create("a");
    const c = await create("c");
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      {
        wrapper,
      },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toEqual([a.key, c.key]));
    const b = await create("b");
    await waitFor(() => expect(result.current.data).toEqual([a.key, b.key, c.key]));
  });

  it("should move a renamed record and drop a deleted one", async () => {
    const a = await create("a");
    const b = await create("b");
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      {
        wrapper,
      },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toEqual([a.key, b.key]));
    await client.ranges.rename(a.key, "c");
    await waitFor(() => expect(result.current.data).toEqual([b.key, a.key]));
    await client.ranges.delete(b.key);
    await waitFor(() => expect(result.current.data).toEqual([a.key]));
  });

  it("should read the next page on more", async () => {
    const a = await create("a");
    const b = await create("b");
    const c = await create("c");
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName(), pageSize: 2 }),
      { wrapper },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toEqual([a.key, b.key]));
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toEqual([a.key, b.key, c.key]));
  });

  it("should read the initial size first, then a page on each more", async () => {
    const a = await create("a");
    const b = await create("b");
    const c = await create("c");
    const d = await create("d");
    const { result } = renderHook(
      () =>
        Flux.useList(ranges, {
          name: "ranges",
          query: byName(),
          pageSize: 1,
          initialSize: 2,
        }),
      { wrapper },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toEqual([a.key, b.key]));
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toEqual([a.key, b.key, c.key]));
    expect(result.current.data).not.toContain(d.key);
  });

  it("should list the records that match a search term", async () => {
    const name = randomWord();
    const match = await create(name);
    await create("other");
    await waitForIndex(name);
    const { result } = renderHook(
      () =>
        Flux.useList(ranges, { name: "ranges", query: byName(), searchDebounce: 0 }),
      { wrapper },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toHaveLength(2));
    act(() => result.current.search(name));
    await waitFor(() => expect(result.current.data).toEqual([match.key]));
    act(() => result.current.search(""));
    await waitFor(() => expect(result.current.data).toHaveLength(2));
  });

  it("should keep the listed records until a search answers", async () => {
    const name = randomWord();
    const match = await create(name);
    await create("other");
    await waitForIndex(name);
    const local = createTestClient();
    const { result } = renderHook(
      () =>
        Flux.useList(ranges, { name: "ranges", query: byName(), searchDebounce: 0 }),
      { wrapper: createSynnaxWrapper({ client: local }) },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toHaveLength(2));
    const { responded, release } = holdResponses(local, "/range/retrieve");
    act(() => result.current.search(name));
    await responded;
    expect(result.current.data).toHaveLength(2);
    expect(result.current.answered).toBe(true);
    release();
    await waitFor(() => expect(result.current.data).toEqual([match.key]));
  });

  it("should hide the records the filter rejects", async () => {
    await create("a");
    const b = await create("b");
    const { result } = renderHook(
      () =>
        Flux.useList(ranges, {
          name: "ranges",
          query: byName(),
          filter: (r) => r.name !== "a",
        }),
      { wrapper },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toEqual([b.key]));
  });

  it("should fetch a subscribed record outside the list", async () => {
    const outside = await client.ranges.create({
      name: "outside",
      timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
    });
    // A client that has read nothing holds no record of the range.
    const reader = createTestClient();
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      {
        wrapper: createSynnaxWrapper({ client: reader }),
      },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.answered).toBe(true));
    expect(result.current.getItem(outside.key)).toBeUndefined();
    const callback = vi.fn();
    const release = result.current.subscribe(callback, outside.key);
    await waitFor(() => expect(callback).toHaveBeenCalled());
    expect(result.current.getItem(outside.key)?.name).toEqual("outside");
    expect(result.current.getItem([outside.key])).toHaveLength(1);
    release();
  });

  it("should read nothing before the first more or search", async () => {
    await create("a");
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      {
        wrapper,
      },
    );
    await act(async () => await sleep.sleep(TimeSpan.milliseconds(50)));
    expect(result.current.answered).toBe(false);
    expect(result.current.data).toEqual([]);
  });

  it("should start the list on a search", async () => {
    const name = randomWord();
    const match = await create(name);
    await waitForIndex(name);
    const { result } = renderHook(
      () =>
        Flux.useList(ranges, { name: "ranges", query: byName(), searchDebounce: 0 }),
      { wrapper },
    );
    act(() => result.current.search(name));
    await waitFor(() => expect(result.current.data).toEqual([match.key]));
  });

  it("should report a failed read and show no records of the last query", async () => {
    await create("a");
    const local = createTestClient();
    const { result } = renderHook(
      () =>
        Flux.useList(ranges, { name: "ranges", query: byName(), searchDebounce: 0 }),
      { wrapper: createSynnaxWrapper({ client: local }) },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toHaveLength(1));
    spyOnSend(local).mockRejectedValue(new Error("boom"));
    act(() => result.current.search(randomWord()));
    await waitFor(() => expect(result.current.variant).toEqual("error"));
    expect(result.current.data).toEqual([]);
    expect(result.current.answered).toBe(false);
  });

  it("should read nothing on more while a read runs", async () => {
    await create("a");
    await create("b");
    const local = createTestClient();
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName(), pageSize: 1 }),
      { wrapper: createSynnaxWrapper({ client: local }) },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toHaveLength(1));
    const send = spyOnSend(local);
    act(() => {
      result.current.more();
      result.current.more();
    });
    await waitFor(() => expect(result.current.data).toHaveLength(2));
    expect(
      send.mock.calls.filter(([target]) => target === "/range/retrieve"),
    ).toHaveLength(1);
  });

  it("should read nothing on more once the list is complete", async () => {
    await create("a");
    const local = createTestClient();
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      { wrapper: createSynnaxWrapper({ client: local }) },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.complete).toBe(true));
    const send = spyOnSend(local);
    act(() => result.current.more());
    await sleep.sleep(TimeSpan.milliseconds(20));
    expect(send).not.toHaveBeenCalled();
  });

  it("should show the cached records on a remount before the first more", async () => {
    const a = await create("a");
    const first = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      { wrapper },
    );
    act(() => first.result.current.more());
    await waitFor(() => expect(first.result.current.data).toEqual([a.key]));
    first.unmount();
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      { wrapper },
    );
    expect(result.current.data).toEqual([a.key]);
    expect(result.current.answered).toBe(true);
  });

  it("should read no record for a zero key", () => {
    const item = vi.fn((key: string) => key);
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", itemQuery: item }),
      { wrapper },
    );
    expect(result.current.getItem("")).toBeUndefined();
    expect(item).not.toHaveBeenCalled();
  });

  it("should be disabled without a query", () => {
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: null }),
      {
        wrapper,
      },
    );
    expect(result.current.variant).toEqual("disabled");
    expect(result.current.data).toEqual([]);
  });

  it("should be disabled without a client", () => {
    const { result } = renderHook(() => Flux.useList(ranges, { name: "ranges" }), {
      wrapper: createSynnaxWrapper({ client: null }),
    });
    expect(result.current.variant).toEqual("disabled");
  });
});
