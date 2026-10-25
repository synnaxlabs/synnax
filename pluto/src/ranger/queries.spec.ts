// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
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

import { Flux } from "@/flux";
import { Ranger } from "@/ranger";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

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
  describe("useList", () => {
    it("should update an item read by key when it changes", async () => {
      const item = await client.ranges.create({
        name: "original",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const { result } = renderHook(
        () => {
          const list = Ranger.useList({ useCachedList: false });
          return Flux.useListItem({
            subscribe: list.subscribe,
            getItem: list.getItem,
            key: item.key,
          });
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current?.name).toEqual("original"));
      await act(async () => {
        await client.ranges.rename(item.key, "updated");
      });
      await waitFor(() => expect(result.current?.name).toEqual("updated"));
    });

    it("should return a list of range keys", async () => {
      const range1 = await client.ranges.create({
        name: "range1",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const range2 = await client.ranges.create({
        name: "range2",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
      });

      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(range1.key);
      expect(result.current.data).toContain(range2.key);
    });

    it("should get individual ranges using getItem", async () => {
      const testRange = await client.ranges.create({
        name: "testRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        color: "#FF5733",
      });

      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() =>
        expect(
          result.current.variant,
          `${result.current.status.message}:${result.current.status.description}`,
        ).toEqual("success"),
      );

      const retrievedRange = result.current.getItem(testRange.key);
      expect(retrievedRange?.key).toEqual(testRange.key);
      expect(retrievedRange?.name).toEqual("testRange");
      expect(retrievedRange?.color).toEqual(color.construct("#FF5733"));
    });

    it("should filter ranges by search term", async () => {
      await client.ranges.create({
        name: "ordinary_range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await client.ranges.create({
        name: "special_range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });

      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve(
          { searchTerm: "special" },
          { signal: controller.signal },
        );
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data.length).toBeGreaterThanOrEqual(1);
        expect(
          result.current.data
            .map((key: ranger.Key) => result.current.getItem(key)?.name)
            .includes("special_range"),
        ).toBe(true);
      });
    });

    it("should handle includeLabels parameter", async () => {
      const testRange = await client.ranges.create({
        name: "labeledRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });

      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ includeLabels: true }, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedRange = result.current.getItem(testRange.key);
      expect(retrievedRange?.key).toEqual(testRange.key);
    });

    it("should handle includeParent parameter", async () => {
      const parentRange = await client.ranges.create({
        name: "parentRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
      });
      const childRange = await client.ranges.create({
        name: "childRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        parent: parentRange,
      });

      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ includeParent: true }, { signal: controller.signal });
      });
      await waitFor(() =>
        expect(
          result.current.variant,
          `${result.current.status.message}:${result.current.status.description}`,
        ).toEqual("success"),
      );

      const retrievedChild = result.current.getItem(childRange.key);
      expect(retrievedChild?.key).toEqual(childRange.key);
    });

    it("should handle pagination with limit and offset", async () => {
      const keys: ranger.Key[] = [];
      for (let i = 0; i < 5; i++) {
        const range = await client.ranges.create({
          name: `paginationRange${i}`,
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        keys.push(range.key);
      }

      const { result } = renderHook(() => Ranger.useList({ initialQuery: { keys } }), {
        wrapper,
      });
      act(() => {
        result.current.retrieve(
          { limit: 2, offset: 1, keys },
          { signal: controller.signal },
        );
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should update the list when a range is created", async () => {
      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      const initialLength = result.current.data.length;

      const newRange = await client.ranges.create({
        name: "newRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });

      await waitFor(() => {
        expect(result.current.data.length).toBeGreaterThan(initialLength);
        expect(result.current.data).toContain(newRange.key);
      });
    });

    it("should update the list when a range is updated", async () => {
      const testRange = await client.ranges.create({
        name: "original",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });

      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.getItem(testRange.key)?.name).toEqual("original");

      await client.ranges.rename(testRange.key, "updated");

      await waitFor(() => {
        expect(result.current.getItem(testRange.key)?.name).toEqual("updated");
      });
    });

    it("should remove range from list when deleted", async () => {
      const testRange = await client.ranges.create({
        name: "toDelete",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });

      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data).toContain(testRange.key);

      await client.ranges.delete(testRange.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testRange.key);
      });
    });

    it("should filter ranges by labels", async () => {
      const label = await client.labels.create({
        name: "Filter Label",
        color: "#000000",
      });
      const r = await client.ranges.create({
        name: "Filter Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await client.labels.label(ranger.ontologyID(r.key), [label.key]);
      const { result } = renderHook(
        () => Ranger.useList({ initialQuery: { hasLabels: [label.key] } }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve(
          { hasLabels: [label.key] },
          { signal: controller.signal },
        );
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(1);
      expect(result.current.data).toContain(r.key);
      // add a new range without the label
      const r2 = await client.ranges.create({
        name: "Unlabeled Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      act(() => {
        result.current.retrieve(
          { hasLabels: [label.key] },
          { signal: controller.signal },
        );
      });
      await waitFor(() => expect(result.current.data).toHaveLength(1));
      expect(result.current.data).not.toContain(r2.key);
      const r3 = await client.ranges.create({
        name: "Labeled Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await client.labels.label(ranger.ontologyID(r3.key), [label.key]);
      act(() => {
        result.current.retrieve(
          { hasLabels: [label.key] },
          { signal: controller.signal },
        );
      });
      await waitFor(() => expect(result.current.data).toHaveLength(2));
      expect(result.current.data).toContain(r3.key);
    });

    it("should handle ranges with custom colors", async () => {
      const coloredRange = await client.ranges.create({
        name: "coloredRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        color: "#E774D0",
      });

      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedRange = result.current.getItem(coloredRange.key);
      expect(retrievedRange?.color).toEqual(color.construct("#E774D0"));
    });

    it("should update range when a label is deleted", async () => {
      const label1 = await client.labels.create({
        name: "labelToDelete",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "labelToKeep",
        color: "#00FF00",
      });
      const testRange = await client.ranges.create({
        name: "rangeWithLabels",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await client.labels.label(ranger.ontologyID(testRange.key), [
        label1.key,
        label2.key,
      ]);

      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ includeLabels: true }, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const rangeBeforeDelete = result.current.getItem(testRange.key);
      expect(rangeBeforeDelete?.labels?.map((l) => l.key)).toContain(label1.key);
      expect(rangeBeforeDelete?.labels?.map((l) => l.key)).toContain(label2.key);

      await client.labels.remove(testRange.ontologyID, [label1.key]);

      await waitFor(() => {
        const rangeAfterDelete = result.current.getItem(testRange.key);
        expect(rangeAfterDelete?.labels?.map((l) => l.key)).not.toContain(label1.key);
        expect(rangeAfterDelete?.labels?.map((l) => l.key)).toContain(label2.key);
      });
    });

    it("should handle ranges with different time spans", async () => {
      const shortRange = await client.ranges.create({
        name: "shortRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.milliseconds(500)),
      });
      const longRange = await client.ranges.create({
        name: "longRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.minutes(5)),
      });

      const { result } = renderHook(() => Ranger.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedShort = result.current.getItem(shortRange.key);
      const retrievedLong = result.current.getItem(longRange.key);

      expect(retrievedShort?.name).toEqual("shortRange");
      expect(retrievedLong?.name).toEqual("longRange");
      expect(retrievedLong?.timeRange.span.milliseconds).toBeGreaterThan(
        retrievedShort?.timeRange.span.milliseconds ?? 0,
      );
    });
  });

  describe("useChildren", () => {
    it("should return a list of child range keys", async () => {
      const parentRange = await client.ranges.create({
        name: "parentRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      });
      const child1 = await client.ranges.create({
        name: "child1",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
        parent: parentRange,
      });
      const child2 = await client.ranges.create({
        name: "child2",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(3)),
        parent: parentRange,
      });

      const { result } = renderHook(() => Ranger.useListChildren(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve(
          { key: parentRange.key },
          { signal: controller.signal },
        );
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(child1.key);
      expect(result.current.data).toContain(child2.key);
    });

    it("should forward the page bounds to the client", async () => {
      const parentRange = await client.ranges.create({
        name: "pagedParent",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      });
      await Promise.all(
        Array.from({ length: 3 }, (_, i) =>
          client.ranges.create({
            name: `pagedChild${i}`,
            timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
            parent: parentRange,
          }),
        ),
      );
      const spy = vi.spyOn(client.ranges.children, "retrieve");
      try {
        const { result } = renderHook(() => Ranger.useListChildren(), { wrapper });
        act(() => {
          result.current.retrieve(
            { key: parentRange.key, limit: 2, offset: 0 },
            { signal: controller.signal },
          );
        });
        await waitFor(() => expect(result.current.variant).toEqual("success"));
        expect(result.current.data).toHaveLength(2);
        expect(spy).toHaveBeenCalledWith({ key: parentRange.key, limit: 2, offset: 0 });
      } finally {
        spy.mockRestore();
      }
    });

    it("should get individual child ranges using getItem", async () => {
      const parentRange = await client.ranges.create({
        name: "parentRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      });
      const childRange = await client.ranges.create({
        name: "testChild",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
        color: "#00FF00",
        parent: parentRange,
      });

      const { result } = renderHook(() => Ranger.useListChildren(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve(
          { key: parentRange.key },
          { signal: controller.signal },
        );
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedChild = result.current.getItem(childRange.key);
      expect(retrievedChild?.key).toEqual(childRange.key);
      expect(retrievedChild?.name).toEqual("testChild");
      expect(retrievedChild?.color).toEqual(color.construct("#00FF00"));
    });

    it("should return empty list for range with no children", async () => {
      const parentRange = await client.ranges.create({
        name: "lonelyParent",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(5)),
      });

      const { result } = renderHook(() => Ranger.useListChildren(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve(
          { key: parentRange.key },
          { signal: controller.signal },
        );
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(0);
    });

    it("should update the list when a child range is created", async () => {
      const parentRange = await client.ranges.create({
        name: "parentRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      });

      const { result } = renderHook(() => Ranger.useListChildren(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve(
          { key: parentRange.key },
          { signal: controller.signal },
        );
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      const initialLength = result.current.data.length;
      expect(initialLength).toEqual(0);

      const newChild = await client.ranges.create({
        name: "newChild",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        parent: parentRange,
      });

      await waitFor(() => {
        expect(result.current.data.length).toBeGreaterThan(initialLength);
        expect(result.current.data).toContain(newChild.key);
      });
    });

    it("should update the list when a child range is updated", async () => {
      const parentRange = await client.ranges.create({
        name: "parentRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      });
      const childRange = await client.ranges.create({
        name: "originalChild",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
        parent: parentRange,
      });

      const { result } = renderHook(() => Ranger.useListChildren(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve(
          { key: parentRange.key },
          { signal: controller.signal },
        );
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.getItem(childRange.key)?.name).toEqual("originalChild");

      await client.ranges.rename(childRange.key, "updatedChild");

      await waitFor(() => {
        expect(result.current.getItem(childRange.key)?.name).toEqual("updatedChild");
      });
    });

    it("should remove child from list when deleted", async () => {
      const parentRange = await client.ranges.create({
        name: "parentRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      });
      const childRange = await client.ranges.create({
        name: "childToDelete",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
        parent: parentRange,
      });

      const { result } = renderHook(() => Ranger.useListChildren(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve(
          { key: parentRange.key },
          { signal: controller.signal },
        );
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data).toContain(childRange.key);

      await client.ranges.delete(childRange.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(childRange.key);
      });
    });

    it("should handle nested parent-child relationships", async () => {
      const grandparentRange = await client.ranges.create({
        name: "grandparent",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(20)),
      });
      const parentRange = await client.ranges.create({
        name: "parent",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(10)),
        parent: grandparentRange,
      });
      const childRange = await client.ranges.create({
        name: "child",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(5)),
        parent: parentRange,
      });

      const { result: grandparentResult } = renderHook(() => Ranger.useListChildren(), {
        wrapper,
      });
      act(() => {
        grandparentResult.current.retrieve(
          { key: grandparentRange.key },
          { signal: controller.signal },
        );
      });
      await waitFor(() => expect(grandparentResult.current.variant).toEqual("success"));
      expect(grandparentResult.current.data).toContain(parentRange.key);
      expect(grandparentResult.current.data).not.toContain(childRange.key);

      const { result: parentResult } = renderHook(() => Ranger.useListChildren(), {
        wrapper,
      });
      act(() => {
        parentResult.current.retrieve(
          { key: parentRange.key },
          { signal: controller.signal },
        );
      });
      await waitFor(() => expect(parentResult.current.variant).toEqual("success"));
      expect(parentResult.current.data).toContain(childRange.key);
      expect(parentResult.current.data).not.toContain(parentRange.key);
    });

    it("should handle multiple levels of hierarchy correctly", async () => {
      const rootRange = await client.ranges.create({
        name: "root",
        timeRange: TimeStamp.now().spanRange(TimeSpan.hours(1)),
      });

      // Create multiple children at the same level
      const children = [];
      for (let i = 0; i < 3; i++) {
        const child = await client.ranges.create({
          name: `level1_child_${i}`,
          timeRange: TimeStamp.now().spanRange(TimeSpan.minutes(10)),
          parent: rootRange,
        });
        children.push(child);
      }

      const { result } = renderHook(() => Ranger.useListChildren(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ key: rootRange.key }, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toHaveLength(3);
      children.forEach((child) => {
        expect(result.current.data).toContain(child.key);
      });
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
        result.current.form.set("name", "newFormRange");
        result.current.form.set("timeRange", timeRange.numeric);
        result.current.form.set("color", "#FF0000");
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
        result.current.form.set("name", "labeledRange");
        result.current.form.set("timeRange", timeRange.numeric);
        result.current.form.set("labels", [label1.key, label2.key]);
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
        result.current.form.set("name", "childRange");
        result.current.form.set("timeRange", childTimeRange.numeric);
        result.current.form.set("parent", parentRange.key);
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
        result.current.form.set("name", "complexRange");
        result.current.form.set("timeRange", complexTimeRange.numeric);
        result.current.form.set("parent", parentRange.key);
        result.current.form.set("labels", [label.key]);
        result.current.form.set("color", color.construct("#654321"));
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
        result.current.form.set("name", "timeRangeTest");
        result.current.form.set("timeRange", initialTimeRange.numeric);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().timeRange).toEqual(initialTimeRange.numeric);
      });

      const modifiedTimeRange = TimeStamp.now().spanRange(TimeSpan.minutes(20));
      act(() => {
        result.current.form.set("timeRange", modifiedTimeRange.numeric);
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
        result.current.form.set("name", "parentChangeTest");
        result.current.form.set("timeRange", timeRange.numeric);
        result.current.form.set("parent", parent1.key);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().parent).toEqual(parent1.key);
      });

      act(() => {
        result.current.form.set("parent", parent2.key);
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
        result.current.form.set("key", "operator");
        result.current.form.set("value", "Ada");
      });
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      expect(await rng.kv.get("operator")).toBe("Ada");
    });
  });

  describe("use cached fast-path", () => {
    it("resolves synchronously from the warm store without suspending", async () => {
      const rng = await client.ranges.create({
        name: "cached_fastpath",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });

      // Warm the flux store (range + labels + parent relationship) through the
      // async retrieve path so the composed cache fast-path can resolve.
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
    act(() => result.current.form.set("name", "Cold flow"));
    expect(result.current.form.get("name").value).toBe("Cold flow");
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(send.mock.calls.map(([target]) => target)).toContain("/range/dispatch");
    expect((await createTestClient().ranges.retrieve(rng.key)).name).toBe("Cold flow");
  });

  it("should send an edit to the time range", async () => {
    const rng = await createRange();
    const { result } = await renderForm(rng.key);
    const next = rng.timeRange.start.spanRange(TimeSpan.minutes(10));
    act(() => result.current.form.set("timeRange", next.numeric));
    await act(async () => expect(await result.current.flush()).toBe(true));
    const stored = await createTestClient().ranges.retrieve(rng.key);
    expect(stored.timeRange.numeric).toEqual(next.numeric);
  });

  it("should hold a time range that ends before it starts", async () => {
    const rng = await createRange();
    const { result } = await renderForm(rng.key);
    const { start, end } = rng.timeRange.numeric;
    act(() => result.current.form.set("timeRange", { start: end, end: start }));
    await act(async () => expect(await result.current.flush()).toBe(false));
    const stored = await createTestClient().ranges.retrieve(rng.key);
    expect(stored.timeRange.numeric).toEqual(rng.timeRange.numeric);
  });

  it("should show a rename from another client and keep the time range", async () => {
    const rng = await createRange();
    const { result } = await renderForm(rng.key);
    const timeRange = result.current.form.get("timeRange").value;
    await act(async () => {
      await client.ranges.rename(rng.key, "Cold flow");
    });
    await waitFor(() => expect(result.current.form.value().name).toBe("Cold flow"));
    expect(result.current.form.get("timeRange").value).toBe(timeRange);
  });
});
