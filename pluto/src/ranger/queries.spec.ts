// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, ranger } from "@synnaxlabs/client";
import { color, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Ontology } from "@/ontology";
import { Ranger } from "@/ranger";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let controller: AbortController;
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    controller = new AbortController();
    wrapper = await createAsyncSynnaxWrapper({
      client,
      excludeFluxStores: [Ontology.RESOURCES_FLUX_STORE_KEY],
    });
  });
  afterEach(() => {
    controller.abort();
  });
  describe("useList", () => {
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
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(1);
      expect(
        result.current.data
          .map((key: ranger.Key) => result.current.getItem(key)?.name)
          .includes("special_range"),
      ).toBe(true);
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
      const childRange = await client.ranges.create(
        {
          name: "childRange",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        },
        { parent: parentRange.ontologyID },
      );

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
      // add a new range with the label
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
      const child1 = await client.ranges.create(
        {
          name: "child1",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
        },
        { parent: parentRange.ontologyID },
      );
      const child2 = await client.ranges.create(
        {
          name: "child2",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(3)),
        },
        { parent: parentRange.ontologyID },
      );

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

    it("should get individual child ranges using getItem", async () => {
      const parentRange = await client.ranges.create({
        name: "parentRange",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      });
      const childRange = await client.ranges.create(
        {
          name: "testChild",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
          color: "#00FF00",
        },
        { parent: parentRange.ontologyID },
      );

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

      const newChild = await client.ranges.create(
        {
          name: "newChild",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        },
        { parent: parentRange.ontologyID },
      );

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
      const childRange = await client.ranges.create(
        {
          name: "originalChild",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
        },
        { parent: parentRange.ontologyID },
      );

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
      const childRange = await client.ranges.create(
        {
          name: "childToDelete",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
        },
        { parent: parentRange.ontologyID },
      );

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
      const parentRange = await client.ranges.create(
        {
          name: "parent",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(10)),
        },
        { parent: grandparentRange.ontologyID },
      );
      const childRange = await client.ranges.create(
        {
          name: "child",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(5)),
        },
        { parent: parentRange.ontologyID },
      );

      // Test grandparent's children
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

      // Test parent's children
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
        const child = await client.ranges.create(
          {
            name: `level1_child_${i}`,
            timeRange: TimeStamp.now().spanRange(TimeSpan.minutes(10)),
          },
          { parent: rootRange.ontologyID },
        );
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

  describe("useForm", () => {
    it("should create a new range", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.minutes(5));

      const { result } = renderHook(() => Ranger.useForm({ query: {} }), {
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

      const { result } = renderHook(() => Ranger.useForm({ query: {} }), {
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

      const { result } = renderHook(() => Ranger.useForm({ query: {} }), {
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

    it("should retrieve and edit existing range", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.minutes(10));
      const existingRange = await client.ranges.create({
        name: "existingRange",
        timeRange,
        color: "#FFFF00",
      });

      const { result } = renderHook(
        () => Ranger.useForm({ query: { key: existingRange.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().name).toEqual("existingRange");
      expect(result.current.form.value().color).toEqual(color.construct("#FFFF00"));
      expect(result.current.form.value().timeRange).toEqual(timeRange.numeric);

      act(() => {
        result.current.form.set("name", "editedRange");
        result.current.form.set("color", "#00FFFF");
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("editedRange");
        expect(result.current.form.value().color).toEqual(color.construct("#00FFFF"));
      });
    });

    it("should retrieve range with existing labels", async () => {
      const label1 = await client.labels.create({
        name: "existingLabel1",
        color: color.construct("#FF00FF"),
      });
      const label2 = await client.labels.create({
        name: "existingLabel2",
        color: color.construct("#FFFF00"),
      });

      const timeRange = TimeStamp.now().spanRange(TimeSpan.minutes(8));
      const existingRange = await client.ranges.create({
        name: "rangeWithLabels",
        timeRange,
      });

      await act(async () => {
        await client.labels.label(existingRange.ontologyID, [label1.key, label2.key]);
      });

      const { result } = renderHook(
        () => Ranger.useForm({ query: { key: existingRange.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().name).toEqual("rangeWithLabels");
      expect(result.current.form.value().labels).toEqual(
        expect.arrayContaining([label1.key, label2.key]),
      );
    });

    it("should retrieve range with parent relationship", async () => {
      const parentRange = await client.ranges.create({
        name: "parentForRetrieval",
        timeRange: TimeStamp.now().spanRange(TimeSpan.hours(2)),
      });
      const childRange = await client.ranges.create(
        {
          name: "childForRetrieval",
          timeRange: TimeStamp.now().spanRange(TimeSpan.minutes(15)),
        },
        { parent: parentRange.ontologyID },
      );

      const { result } = renderHook(
        () => Ranger.useForm({ query: { key: childRange.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().name).toEqual("childForRetrieval");
      expect(result.current.form.value().parent).toEqual(parentRange.key);
    });

    it("should update form when range is updated externally", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.minutes(7));
      const testRange = await client.ranges.create({
        name: "externalUpdate",
        timeRange,
      });

      const { result } = renderHook(
        () => Ranger.useForm({ query: { key: testRange.key } }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.form.value().name).toEqual("externalUpdate");

      await client.ranges.rename(testRange.key, "externallyUpdated");

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("externallyUpdated");
      });
    });

    it("should update form when labels are added externally", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.minutes(4));
      const testRange = await client.ranges.create({
        name: "labelUpdateTest",
        timeRange,
      });

      const { result } = renderHook(
        () => Ranger.useForm({ query: { key: testRange.key } }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const initialLabels = result.current.form.value().labels;

      const newLabel = await act(async () => {
        const newLabel = await client.labels.create({
          name: "externalLabel",
          color: "#FF5500",
        });
        await client.labels.label(testRange.ontologyID, [newLabel.key]);
        return newLabel;
      });

      await waitFor(() => {
        expect(result.current.form.value().labels).toHaveLength(
          initialLabels.length + 1,
        );
        expect(result.current.form.value().labels).toContain(newLabel.key);
      });
    });

    it("should update form when labels are removed externally", async () => {
      const label1 = await client.labels.create({
        name: "labelToRemove",
        color: "#AA0000",
      });
      const label2 = await client.labels.create({
        name: "labelToKeep",
        color: "#00AA00",
      });

      const timeRange = TimeStamp.now().spanRange(TimeSpan.minutes(6));
      const testRange = await client.ranges.create({
        name: "labelRemovalTest",
        timeRange,
      });

      await client.labels.label(testRange.ontologyID, [label1.key, label2.key]);

      const { result } = renderHook(
        () => Ranger.useForm({ query: { key: testRange.key } }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        const labels = result.current.form.value().labels;
        expect(labels).toContain(label1.key);
        expect(labels).toContain(label2.key);
      });

      await client.labels.remove(testRange.ontologyID, [label1.key]);

      await waitFor(() => {
        const labels = result.current.form.value().labels;
        expect(labels).not.toContain(label1.key);
        expect(labels).toContain(label2.key);
      });
    });

    it("should update form when parent is changed externally", async () => {
      const originalParent = await client.ranges.create({
        name: "originalParent",
        timeRange: TimeStamp.now().spanRange(TimeSpan.hours(1)),
      });
      const newParent = await client.ranges.create({
        name: "newParent",
        timeRange: TimeStamp.now().spanRange(TimeSpan.hours(2)),
      });
      const childRange = await client.ranges.create(
        {
          name: "childForParentChange",
          timeRange: TimeStamp.now().spanRange(TimeSpan.minutes(30)),
        },
        { parent: originalParent.ontologyID },
      );

      const { result } = renderHook(
        () => Ranger.useForm({ query: { key: childRange.key } }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.form.value().parent).toEqual(originalParent.key);

      await client.ontology.moveChildren(
        originalParent.ontologyID,
        newParent.ontologyID,
        childRange.ontologyID,
      );

      await waitFor(() => {
        expect(result.current.form.value().parent).toEqual(newParent.key);
      });
    });

    it("should handle form with default values", async () => {
      const { result } = renderHook(() => Ranger.useForm({ query: {} }), {
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

      const { result } = renderHook(() => Ranger.useForm({ query: {} }), {
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
      const { result } = renderHook(() => Ranger.useForm({ query: {} }), {
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

      const { result } = renderHook(() => Ranger.useForm({ query: {} }), {
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
});
