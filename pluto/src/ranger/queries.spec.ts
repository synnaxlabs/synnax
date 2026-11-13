// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ranger } from "@synnaxlabs/client";
import {
  createTestClient,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { color, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import {
  createElement,
  type PropsWithChildren,
  type ReactElement,
  Suspense,
} from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Ranger } from "@/ranger";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const path = Ranger.formPath;
const created = Ranger.createFormPath;
const pair = Ranger.kvPairFormPath;

const client = createTestClient();

describe("queries", () => {
  let controller: AbortController;
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    controller = new AbortController();
    wrapper = await createAsyncSynnaxWrapper({ client });
  });
  afterEach(() => {
    controller.abort();
  });
  describe("useResultMetaData", () => {
    it("should follow a pair set on the range", async () => {
      const rng = await client.ranges.create({
        name: "metadata",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const { result } = renderHook(
        () => Ranger.useResultMetaData({ rangeKey: rng.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.data).toEqual([]));
      await client.ranges.getKV(rng.key).set("key", "value");
      await waitFor(() =>
        expect(result.current.data).toEqual([
          { key: "key", value: "value", range: rng.key },
        ]),
      );
    });
  });

  describe("useResultAncestors", () => {
    it("should map each range to its parent up to the top", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));
      const top = await client.ranges.create({ name: "top", timeRange });
      const middle = await client.ranges.create({
        name: "middle",
        timeRange,
        parent: top.key,
      });
      const bottom = await client.ranges.create({
        name: "bottom",
        timeRange,
        parent: middle.key,
      });
      const { result } = renderHook(
        () => Ranger.useResultAncestors({ keys: [bottom.key] }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.data).toEqual({
          [bottom.key]: middle.key,
          [middle.key]: top.key,
          [top.key]: null,
        }),
      );
    });

    it("should follow a move of a range above to a parent it lacks", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));
      const first = await client.ranges.create({ name: "first", timeRange });
      const middle = await client.ranges.create({
        name: "middle",
        timeRange,
        parent: first.key,
      });
      const bottom = await client.ranges.create({
        name: "bottom",
        timeRange,
        parent: middle.key,
      });
      const second = await client.ranges.create({ name: "second", timeRange });
      const reader = createTestClient();
      const { result } = renderHook(
        () => Ranger.useResultAncestors({ keys: [bottom.key] }),
        { wrapper: await createAsyncSynnaxWrapper({ client: reader }) },
      );
      await waitFor(() => expect(result.current.data?.[middle.key]).toEqual(first.key));
      await waitForStreamLive(reader.connection);
      await client.ranges.dispatch(
        middle.key,
        ranger.set({ path: ["parent"], value: second.key }),
      );
      await waitFor(() =>
        expect(result.current.data).toEqual({
          [bottom.key]: middle.key,
          [middle.key]: second.key,
          [second.key]: null,
        }),
      );
    });
  });

  describe("useDelete", () => {
    it("should delete a range", async () => {
      const rng = await client.ranges.create({
        name: "delete_hook",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const { result } = renderHook(() => Ranger.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(rng.key);
      });
      await expect(client.ranges.retrieve(rng.key)).rejects.toThrow();
    });

    it("should run afterOptimistic before the delete commits", async () => {
      const rng = await client.ranges.create({
        name: "delete_hook_optimistic",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const order: string[] = [];
      const { result } = renderHook(
        () =>
          Ranger.useDelete({
            afterOptimistic: ({ data }) => {
              order.push(`optimistic:${data as ranger.Key}`);
            },
            afterSuccess: () => {
              order.push("success");
            },
          }),
        { wrapper },
      );
      await act(async () => {
        await result.current.updateAsync(rng.key);
      });
      expect(order).toEqual([`optimistic:${rng.key}`, "success"]);
    });
  });

  describe("useSetTimeRange", () => {
    const start = TimeStamp.seconds(1_700_000_000);
    const createRange = async (name: string) =>
      await client.ranges.create({
        name,
        timeRange: start.spanRange(TimeSpan.seconds(1)),
      });

    it("should set the time range of a range", async () => {
      const rng = await createRange("set_time_range_hook");
      const timeRange = start.spanRange(TimeSpan.seconds(5));
      const { result } = renderHook(() => Ranger.useSetTimeRange(), { wrapper });
      await act(async () => {
        expect(
          await result.current.updateAsync({
            key: rng.key,
            timeRange: timeRange.numeric,
          }),
        ).toBe(true);
      });
      expect((await client.ranges.retrieve(rng.key)).timeRange).toEqual(timeRange);
    });

    it("should reject an end before the start", async () => {
      const rng = await createRange("set_time_range_hook_reversed");
      const { result } = renderHook(() => Ranger.useSetTimeRange(), { wrapper });
      await act(async () => {
        expect(
          await result.current.updateAsync({
            key: rng.key,
            timeRange: { start: rng.timeRange.end.nanoseconds, end: start.nanoseconds },
          }),
        ).toBe(false);
      });
      expect(result.current.status.description).toContain(
        "time_range.start cannot be after time_range.end",
      );
      expect((await client.ranges.retrieve(rng.key)).timeRange).toEqual(rng.timeRange);
    });
  });

  describe("useSetColor", () => {
    it("should set the color of a range", async () => {
      const rng = await client.ranges.create({
        name: "set_color_hook",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const { result } = renderHook(() => Ranger.useSetColor(), { wrapper });
      await act(async () => {
        expect(
          await result.current.updateAsync({
            key: rng.key,
            color: color.construct("#ff0000"),
          }),
        ).toBe(true);
      });
      const { color: stored } = await client.ranges.retrieve(rng.key);
      expect(stored == null ? undefined : color.hex(stored)).toEqual("#ff0000");
    });

    it("should clear the color of a range", async () => {
      const rng = await client.ranges.create({
        name: "clear_color_hook",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        color: "#00ff00",
      });
      const { result } = renderHook(() => Ranger.useSetColor(), { wrapper });
      await act(async () => {
        expect(await result.current.updateAsync({ key: rng.key })).toBe(true);
      });
      expect((await client.ranges.retrieve(rng.key)).color).toBeUndefined();
    });
  });

  describe("useUpdateKV", () => {
    it("should set a pair on the range", async () => {
      const rng = await client.ranges.create({
        name: "set_kv_hook",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const { result } = renderHook(() => Ranger.useUpdateKV(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({
          rangeKey: rng.key,
          range: rng.key,
          key: "foo",
          value: "bar",
        });
      });
      expect(await rng.kv.list()).toEqual({ foo: "bar" });
    });

    it("should run afterOptimistic before the set commits", async () => {
      const rng = await client.ranges.create({
        name: "set_kv_hook_optimistic",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const order: string[] = [];
      const { result } = renderHook(
        () =>
          Ranger.useUpdateKV({
            afterOptimistic: ({ data }) => {
              order.push(`optimistic:${data.key}`);
            },
            afterSuccess: () => {
              order.push("success");
            },
          }),
        { wrapper },
      );
      await act(async () => {
        await result.current.updateAsync({
          rangeKey: rng.key,
          range: rng.key,
          key: "foo",
          value: "bar",
        });
      });
      expect(order).toEqual(["optimistic:foo", "success"]);
    });
  });

  describe("useDeleteKV", () => {
    it("should delete a pair from the range", async () => {
      const rng = await client.ranges.create({
        name: "delete_kv_hook",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await rng.kv.set("foo", "bar");
      const { result } = renderHook(() => Ranger.useDeleteKV(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ rangeKey: rng.key, key: "foo" });
      });
      expect(await rng.kv.list()).toEqual({});
    });

    it("should run afterOptimistic before the delete commits", async () => {
      const rng = await client.ranges.create({
        name: "delete_kv_hook_optimistic",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await rng.kv.set("foo", "bar");
      const order: string[] = [];
      const { result } = renderHook(
        () =>
          Ranger.useDeleteKV({
            afterOptimistic: ({ data }) => {
              order.push(`optimistic:${data.key}`);
            },
            afterSuccess: () => {
              order.push("success");
            },
          }),
        { wrapper },
      );
      await act(async () => {
        await result.current.updateAsync({ rangeKey: rng.key, key: "foo" });
      });
      expect(order).toEqual(["optimistic:foo", "success"]);
    });
  });

  describe("useCreateForm", () => {
    it("should create a new range", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.minutes(5));

      const { result } = renderHook(() => Ranger.useCreateForm({ query: null }), {
        wrapper,
      });

      act(() => {
        result.current.form.set(created.name, "newFormRange");
        result.current.form.set(created.timeRange, timeRange.numeric);
        result.current.form.set(created.color, color.construct("#FF0000"));
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("newFormRange");
        expect(result.current.form.value().timeRange).toEqual(timeRange.numeric);
        expect(result.current.form.value().color).toEqual(color.construct("#FF0000"));
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).not.toEqual("");
      });
    });

    it("should create a new range with labels", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.minutes(3));
      const label1 = await client.labels.create({
        name: "testLabel1",
        color: "#00FF00",
      });
      const label2 = await client.labels.create({
        name: "testLabel2",
        color: "#0000FF",
      });

      const { result } = renderHook(() => Ranger.useCreateForm({ query: null }), {
        wrapper,
      });

      act(() => {
        result.current.form.set(created.name, "labeledRange");
        result.current.form.set(created.timeRange, timeRange.numeric);
        result.current.form.set(created.labels, [label1.key, label2.key]);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("labeledRange");
        expect(result.current.form.value().labels).toEqual([label1.key, label2.key]);
        expect(result.current.form.value().key).toBeDefined();
      });
    });

    it("should create a new range with a parent", async () => {
      const parentRange = await client.ranges.create({
        name: "parentRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.hours(1)),
      });
      const childTimeRange = TimeStamp.now().spanRange(TimeSpan.minutes(30));

      const { result } = renderHook(() => Ranger.useCreateForm({ query: null }), {
        wrapper,
      });

      act(() => {
        result.current.form.set(created.name, "childRange");
        result.current.form.set(created.timeRange, childTimeRange.numeric);
        result.current.form.set(created.parent, parentRange.key);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("childRange");
        expect(result.current.form.value().parent).toEqual(parentRange.key);
        expect(result.current.form.value().key).toBeDefined();
      });
    });

    it("should handle form with default values", async () => {
      const { result } = renderHook(() => Ranger.useCreateForm({ query: null }), {
        wrapper,
      });

      expect(result.current.form.value().name).toEqual("");
      expect(result.current.form.value().key).toBeUndefined();
      expect(result.current.form.value().labels).toEqual([]);
      expect(result.current.form.value().parent).toEqual("");
      expect(result.current.form.value().timeRange).toEqual({ start: 0, end: 0 });
    });

    it("should handle complex range operations", async () => {
      const parentRange = await client.ranges.create({
        name: "complexParent",
        timeRange: TimeStamp.now().spanRange(TimeSpan.hours(3)),
      });

      const label = await client.labels.create({
        name: "complexLabel",
        color: "#123456",
      });

      const { result } = renderHook(() => Ranger.useCreateForm({ query: null }), {
        wrapper,
      });

      const complexTimeRange = TimeStamp.now().spanRange(TimeSpan.minutes(45));

      act(() => {
        result.current.form.set(created.name, "complexRange");
        result.current.form.set(created.timeRange, complexTimeRange.numeric);
        result.current.form.set(created.parent, parentRange.key);
        result.current.form.set(created.labels, [label.key]);
        result.current.form.set(created.color, color.construct("#654321"));
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("complexRange");
        expect(result.current.form.value().parent).toEqual(parentRange.key);
        expect(result.current.form.value().labels).toContain(label.key);
        expect(result.current.form.value().color).toEqual(color.construct("#654321"));
        expect(result.current.form.value().timeRange).toEqual(complexTimeRange.numeric);
        expect(result.current.form.value().key).toBeDefined();
      });
    });

    it("should handle time range modifications", async () => {
      const initialTimeRange = TimeStamp.now().spanRange(TimeSpan.minutes(10));
      const { result } = renderHook(() => Ranger.useCreateForm({ query: null }), {
        wrapper,
      });

      act(() => {
        result.current.form.set(created.name, "timeRangeTest");
        result.current.form.set(created.timeRange, initialTimeRange.numeric);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().timeRange).toEqual(initialTimeRange.numeric);
      });

      const modifiedTimeRange = TimeStamp.now().spanRange(TimeSpan.minutes(20));
      act(() => {
        result.current.form.set(created.timeRange, modifiedTimeRange.numeric);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().timeRange).toEqual(
          modifiedTimeRange.numeric,
        );
      });
    });

    it("should handle parent relationship changes", async () => {
      const parent1 = await client.ranges.create({
        name: "parent1",
        timeRange: TimeStamp.now().spanRange(TimeSpan.hours(1)),
      });
      const parent2 = await client.ranges.create({
        name: "parent2",
        timeRange: TimeStamp.now().spanRange(TimeSpan.hours(2)),
      });

      const { result } = renderHook(() => Ranger.useCreateForm({ query: null }), {
        wrapper,
      });

      const timeRange = TimeStamp.now().spanRange(TimeSpan.minutes(30));

      act(() => {
        result.current.form.set(created.name, "parentChangeTest");
        result.current.form.set(created.timeRange, timeRange.numeric);
        result.current.form.set(created.parent, parent1.key);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().parent).toEqual(parent1.key);
      });

      act(() => {
        result.current.form.set(created.parent, parent2.key);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().parent).toEqual(parent2.key);
      });
    });
  });

  describe("useCreateKVPairForm", () => {
    it("should set the pair on the range when it saves", async () => {
      const rng = await client.ranges.create({
        name: "kv_create",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const { result } = renderHook(
        () =>
          Ranger.useCreateKVPairForm({
            query: null,
            initialValues: { key: "", value: "", range: rng.key },
          }),
        { wrapper },
      );
      act(() => {
        result.current.form.set(pair.key, "operator");
        result.current.form.set(pair.value, "Ada");
      });
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      expect(await rng.kv.get("operator")).toBe("Ada");
    });

    it("should fail with Key is required for a blank key", async () => {
      const rng = await client.ranges.create({
        name: "kv_blank",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const { result } = renderHook(
        () =>
          Ranger.useCreateKVPairForm({
            query: null,
            initialValues: { key: "", value: "", range: rng.key },
          }),
        { wrapper },
      );
      act(() => {
        result.current.form.set(pair.key, "   ");
        result.current.form.set(pair.value, "Ada");
      });
      await act(async () => expect(await result.current.saveAsync()).toBe(false));
      expect(result.current.form.get(pair.key).status.message).toBe("Key is required");
      expect(await rng.kv.list()).toEqual({});
    });

    it("should trim the key before it saves", async () => {
      const rng = await client.ranges.create({
        name: "kv_trim",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const { result } = renderHook(
        () =>
          Ranger.useCreateKVPairForm({
            query: null,
            initialValues: { key: "", value: "", range: rng.key },
          }),
        { wrapper },
      );
      act(() => {
        result.current.form.set(pair.key, "  operator  ");
        result.current.form.set(pair.value, "Ada");
      });
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      expect(await rng.kv.list()).toEqual({ operator: "Ada" });
    });
  });

  describe("use cached fast-path", () => {
    it("resolves synchronously from the warm store without suspending", async () => {
      const rng = await client.ranges.create({
        name: "cached_fastpath",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });

      // Warm the flux store through the async retrieve path so the cache fast-path can
      // resolve.
      const warm = await renderHookSuspended(() => Ranger.use({ key: rng.key }), {
        wrapper,
      });
      await waitFor(() => expect(warm.result.current).toBeDefined());

      const Display = (): ReactElement => {
        const range = Ranger.use({ key: rng.key });
        return createElement("span", { "data-testid": "name" }, range.name);
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          createElement(
            wrapper,
            null,
            createElement(
              Suspense,
              { fallback: createElement("span", { "data-testid": "fallback" }) },
              createElement(Display),
            ),
          ),
        );
      });

      expect(utils.queryByTestId("fallback")).toBeNull();
      expect(utils.queryByTestId("name")?.textContent).toBe("cached_fastpath");
    });
  });
});

describe("useLabels", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
  });

  const keysOf = (labels: label.Label[] | undefined) =>
    labels?.map(({ key }) => key).toSorted();

  it("should resolve the labels of a range and follow its label edits", async () => {
    const [first, second] = await client.labels.create([
      { name: "first", color: "#FF0000" },
      { name: "second", color: "#00FF00" },
    ]);
    const rng = await client.ranges.create({
      name: "labeled",
      timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      labels: [first.key],
    });
    const { result } = renderHook(() => Ranger.useLabels(rng.key), { wrapper });
    await waitFor(() => expect(keysOf(result.current)).toEqual([first.key]));
    await client.labels.label(rng.resourceID, [second.key]);
    await waitFor(() =>
      expect(keysOf(result.current)).toEqual([first.key, second.key].toSorted()),
    );
  });

  it("should show the new name of a label the range holds", async () => {
    const lbl = await client.labels.create({ name: "original", color: "#FF0000" });
    const rng = await client.ranges.create({
      name: "labeled",
      timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      labels: [lbl.key],
    });
    const { result } = renderHook(() => Ranger.useLabels(rng.key), { wrapper });
    await waitFor(() => expect(result.current?.[0]?.name).toEqual("original"));
    await client.labels.dispatch(
      lbl.key,
      label.set({ path: ["name"], value: "renamed" }),
    );
    await waitFor(() => expect(result.current?.[0]?.name).toEqual("renamed"));
  });

  it("should answer a range with no labels with an empty list", async () => {
    const rng = await client.ranges.create({
      name: "unlabeled",
      timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
    });
    const { result } = renderHook(() => Ranger.useLabels(rng.key), { wrapper });
    await waitFor(() => expect(result.current).toEqual([]));
  });
});

describe("useForm", () => {
  const formClient = createTestClient();
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client: formClient });
    await waitForStreamLive(formClient.connection);
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  const createRange = async (): Promise<ranger.Range> =>
    await client.ranges.create({
      name: "Hot fire",
      timeRange: TimeStamp.now().spanRange(TimeSpan.minutes(5)),
    });

  const renderForm = async (key: ranger.Key) =>
    await renderHookSuspended(() => Ranger.useForm({ query: { key } }), { wrapper });

  it("should show the cached range", async () => {
    const rng = await createRange();
    const { result } = await renderForm(rng.key);
    expect(result.current.form.value()).toEqual({
      name: "Hot fire",
      timeRange: rng.timeRange.numeric,
    });
  });

  it("should send an edit to the name as a path action", async () => {
    const rng = await createRange();
    const { result } = await renderForm(rng.key);
    const send = spyOnSend(formClient);
    act(() => result.current.form.set(path.name, "Cold flow"));
    expect(result.current.form.get(path.name).value).toBe("Cold flow");
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(send.mock.calls.map(([target]) => target)).toContain("/range/dispatch");
    expect((await createTestClient().ranges.retrieve(rng.key)).name).toBe("Cold flow");
  });

  it("should send an edit to the time range", async () => {
    const rng = await createRange();
    const { result } = await renderForm(rng.key);
    const next = rng.timeRange.start.spanRange(TimeSpan.minutes(10));
    act(() => result.current.form.set(path.timeRange, next.numeric));
    await act(async () => expect(await result.current.flush()).toBe(true));
    const stored = await createTestClient().ranges.retrieve(rng.key);
    expect(stored.timeRange.numeric).toEqual(next.numeric);
  });

  it("should hold a time range that ends before it starts", async () => {
    const rng = await createRange();
    const { result } = await renderForm(rng.key);
    const { start, end } = rng.timeRange.numeric;
    act(() => result.current.form.set(path.timeRange, { start: end, end: start }));
    await act(async () => expect(await result.current.flush()).toBe(false));
    const stored = await createTestClient().ranges.retrieve(rng.key);
    expect(stored.timeRange.numeric).toEqual(rng.timeRange.numeric);
  });

  it("should show a rename from another client and keep the time range", async () => {
    const rng = await createRange();
    const { result } = await renderForm(rng.key);
    const timeRange = result.current.form.get(path.timeRange).value;
    await act(async () => {
      await client.ranges.rename(rng.key, "Cold flow");
    });
    await waitFor(() => expect(result.current.form.value().name).toBe("Cold flow"));
    expect(result.current.form.get(path.timeRange).value).toBe(timeRange);
  });
});
