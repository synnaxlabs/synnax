// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, ranger } from "@synnaxlabs/client";
import {
  createTestClient,
  holdResponses,
  rejectRequests,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { List } from "@synnaxlabs/lyra/list";
import { color, sleep, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { act, render, renderHook, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, useEffect, useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });
const SynnaxWrapper = wrapper;
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

  const byName = (): ranger.RetrieveRequest => ({
    where: { labels: [lbl.key] },
    order: { field: "name", descending: false },
  });

  // A late change signal from a create reads that one range by key. Only a read in the
  // list's order is a list read.
  const listReads = (send: ReturnType<typeof spyOnSend>) =>
    send.mock.calls.filter(
      ([target, req]) =>
        target === "/range/retrieve" &&
        ranger.retrieveRequestZ.parse(req).order != null,
    );

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

  it("should render again only the item whose record changed", async () => {
    const a = await create("a");
    const b = await create("b");
    const renders = new Map<ranger.Key, number>();
    const Item = ({ itemKey }: { itemKey: ranger.Key }): ReactElement => {
      const range = List.useItem<ranger.Key, ranger.Range>(itemKey);
      renders.set(itemKey, (renders.get(itemKey) ?? 0) + 1);
      return <span>{range?.name}</span>;
    };
    const renderItem = ({ key }: { key: ranger.Key }) => (
      <Item key={key} itemKey={key} />
    );
    const Ranges = (): ReactElement => {
      const { data, getItem, subscribe, more } = Flux.useList(ranges, {
        name: "ranges",
        query: byName(),
      });
      useEffect(more, [more]);
      return (
        <List.Frame data={data} getItem={getItem} subscribe={subscribe}>
          <List.Scroll>
            <List.Items<ranger.Key>>{renderItem}</List.Items>
          </List.Scroll>
        </List.Frame>
      );
    };
    render(<Ranges />, { wrapper });
    await screen.findByText("b");
    const rendersOfB = renders.get(b.key);
    await client.ranges.rename(a.key, "a2");
    await screen.findByText("a2");
    expect(renders.get(b.key)).toBe(rendersOfB);
  });

  describe("getItem", () => {
    // The Core alone checks a name pattern, so the list fetches again on each change.
    const byPattern = (): ranger.RetrieveRequest => ({
      where: { and: [{ labels: [lbl.key] }, { name: { regex: "." } }] },
      order: { field: "name", descending: false },
    });

    interface FrameControl {
      getItem?: Flux.UseListReturn<ranger.Key, ranger.Range>["getItem"];
      hide: (key: ranger.Key | null) => void;
    }

    const createFrameWrapper = (query: ranger.RetrieveRequest) => {
      const control: FrameControl = { hide: () => {} };
      const Frame = ({ children }: PropsWithChildren): ReactElement => {
        const [hidden, setHidden] = useState<ranger.Key | null>(null);
        control.hide = setHidden;
        const { data, getItem, subscribe, more } = Flux.useList(ranges, {
          name: "ranges",
          query,
          filter: hidden == null ? undefined : (r) => r.key !== hidden,
        });
        control.getItem = getItem;
        useEffect(more, [more]);
        return (
          <List.Frame data={data} getItem={getItem} subscribe={subscribe}>
            {children}
          </List.Frame>
        );
      };
      const frameWrapper = ({ children }: PropsWithChildren): ReactElement => (
        <SynnaxWrapper>
          <Frame>{children}</Frame>
        </SynnaxWrapper>
      );
      return { control, frameWrapper };
    };

    it("should render again only the reader of the renamed record", async () => {
      const a = await create("a");
      const b = await create("b");
      const { control, frameWrapper } = createFrameWrapper(byPattern());
      let renders = 0;
      const { result } = renderHook(
        () => {
          renders += 1;
          return List.useItem<ranger.Key, ranger.Range>(b.key);
        },
        { wrapper: frameWrapper },
      );
      await waitFor(() => expect(result.current?.name).toEqual("b"));
      const send = spyOnSend(client);
      const before = renders;
      await client.ranges.rename(a.key, "a2");
      await waitFor(() => expect(control.getItem?.(a.key)?.name).toEqual("a2"));
      // The Core alone can place the renamed range, so the list fetches again.
      await waitFor(() => expect(listReads(send)).toHaveLength(1));
      const read = send.mock.calls.findIndex((call) => listReads(send)[0] === call);
      await send.mock.results[read].value;
      await sleep.sleep(TimeSpan.milliseconds(20));
      expect(renders).toEqual(before);
      await client.ranges.rename(b.key, "b2");
      await waitFor(() => expect(result.current?.name).toEqual("b2"));
      send.mockRestore();
    });

    it("should keep its identity through changes and searches", async () => {
      const name = randomWord();
      const match = await create(name);
      const b = await create("b");
      await waitForIndex(name);
      const { result } = renderHook(
        () =>
          Flux.useList(ranges, { name: "ranges", query: byName(), searchDebounce: 0 }),
        { wrapper },
      );
      const { getItem } = result.current;
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([b.key, match.key]));
      await client.ranges.rename(b.key, "b2");
      await waitFor(() => expect(getItem(b.key)?.name).toEqual("b2"));
      act(() => result.current.search(name));
      await waitFor(() => expect(result.current.data).toEqual([match.key]));
      act(() => result.current.search(""));
      await waitFor(() => expect(result.current.data).toHaveLength(2));
      expect(result.current.getItem).toBe(getItem);
    });

    it("should have its readers read again when the filter changes", async () => {
      await create("a");
      const b = await create("b");
      const { control, frameWrapper } = createFrameWrapper(byName());
      const { result } = renderHook(
        () => List.useItem<ranger.Key, ranger.Range>(b.key),
        { wrapper: frameWrapper },
      );
      await waitFor(() => expect(result.current?.name).toEqual("b"));
      act(() => control.hide(b.key));
      expect(result.current).toBeUndefined();
    });
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

  it("should report the search term that its records answer", async () => {
    const name = randomWord();
    const match = await create(name);
    await create("other");
    await waitForIndex(name);
    const local = createTestClient();
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      { wrapper: createSynnaxWrapper({ client: local }) },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toHaveLength(2));
    const { responded, release } = holdResponses(local, "/range/retrieve");
    act(() => result.current.search(name));
    expect(result.current.searchTerm).toEqual("");
    await responded;
    expect(result.current.data).toHaveLength(2);
    expect(result.current.searchTerm).toEqual("");
    release();
    await waitFor(() => expect(result.current.data).toEqual([match.key]));
    expect(result.current.searchTerm).toEqual(name);
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

  it("should read the first page of an eager list as it mounts, and no other", async () => {
    const a = await create("a");
    const b = await create("b");
    await create("c");
    const { result } = renderHook(
      () =>
        Flux.useList(ranges, {
          name: "ranges",
          query: byName(),
          pageSize: 2,
          eager: true,
        }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.data).toEqual([a.key, b.key]));
    act(() => result.current.more());
    await act(async () => await sleep.sleep(TimeSpan.milliseconds(50)));
    expect(result.current.data).toEqual([a.key, b.key]);
    expect(result.current.complete).toBe(false);
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

  it("should show no records of the last query on a failed read or its retry", async () => {
    await create("a");
    const local = createTestClient();
    const { result } = renderHook(
      () =>
        Flux.useList(ranges, { name: "ranges", query: byName(), searchDebounce: 0 }),
      { wrapper: createSynnaxWrapper({ client: local }) },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.data).toHaveLength(1));
    // A change to a range from another spec makes the client read that range by key,
    // and a failed read logs an error. Only the search fails.
    rejectRequests(
      local,
      (target, req) =>
        target === "/range/retrieve" &&
        ranger.retrieveRequestZ.parse(req).searchTerm != null,
      new Error("boom"),
    );
    act(() => result.current.search(randomWord()));
    await waitFor(() => expect(result.current.variant).toEqual("error"));
    expect(result.current.data).toEqual([]);
    expect(result.current.answered).toBe(false);
    act(() => result.current.more());
    expect(result.current.variant).toEqual("loading");
    expect(result.current.data).toEqual([]);
    await waitFor(() => expect(result.current.variant).toEqual("error"));
  });

  it("should read the first page again on more after it fails", async () => {
    const a = await create("a");
    const local = createTestClient();
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      { wrapper: createSynnaxWrapper({ client: local }) },
    );
    const reject = rejectRequests(
      local,
      (target, req) =>
        target === "/range/retrieve" &&
        ranger.retrieveRequestZ.parse(req).order != null,
      new Error("boom"),
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.variant).toEqual("error"));
    reject.mockRestore();
    act(() => result.current.more());
    await waitFor(() => {
      expect(result.current.variant).toEqual("success");
      expect(result.current.data).toEqual([a.key]);
    });
  });

  it("should read nothing on more while a read runs", async () => {
    await create("a");
    await create("b");
    const local = createTestClient();
    // A list read before the stream is live reads again when the stream goes live.
    await waitForStreamLive(local.connection);
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
    expect(listReads(send)).toHaveLength(1);
  });

  it("should read nothing on more once the list is complete", async () => {
    await create("a");
    const local = createTestClient();
    // A list read before the stream is live reads again when the stream goes live.
    await waitForStreamLive(local.connection);
    const { result } = renderHook(
      () => Flux.useList(ranges, { name: "ranges", query: byName() }),
      { wrapper: createSynnaxWrapper({ client: local }) },
    );
    act(() => result.current.more());
    await waitFor(() => expect(result.current.complete).toBe(true));
    const send = spyOnSend(local);
    act(() => result.current.more());
    await sleep.sleep(TimeSpan.milliseconds(20));
    expect(listReads(send)).toHaveLength(0);
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
