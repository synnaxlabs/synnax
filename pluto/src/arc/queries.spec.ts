// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Arc } from "@/arc";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("Arc queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  beforeEach(() => {
    controller = new AbortController();
  });

  afterEach(() => {
    controller.abort();
  });

  describe("useList", () => {
    it("should return a list of arcs", async () => {
      const arc1 = await client.arcs.create({
        name: "arc1",
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });
      const arc2 = await client.arcs.create({
        name: "arc2",
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });

      const { result } = renderHook(() => Arc.useList({}), { wrapper });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(arc1.key);
      expect(result.current.data).toContain(arc2.key);

      const retrievedArc1 = result.current.getItem(arc1.key);
      expect(retrievedArc1?.name).toBe("arc1");
      const retrievedArc2 = result.current.getItem(arc2.key);
      expect(retrievedArc2?.name).toBe("arc2");
    });

    it("should update when a new arc is added", async () => {
      const { result } = renderHook(() => Arc.useList({}), { wrapper });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const initialLength = result.current.data.length;

      await act(async () => {
        await client.arcs.create({
          name: "new-arc",
          version: "1.0.0",
          graph: { nodes: [], edges: [] },
          text: { raw: "" },
          });
      });

      await waitFor(() => {
        expect(result.current.data.length).toBe(initialLength + 1);
      });

      const newArc = result.current.data
        .map((key) => result.current.getItem(key))
        .find((arc) => arc?.name === "new-arc");
      expect(newArc).toBeDefined();
      expect(newArc?.name).toBe("new-arc");
    });

    it("should update when an arc is modified", async () => {
      const testArc = await client.arcs.create({
        name: "original-name",
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });

      const { result } = renderHook(() => Arc.useList({}), { wrapper });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.getItem(testArc.key)?.name).toEqual("original-name");

      await act(async () => {
        await client.arcs.create({
          ...testArc,
          name: "updated-name",
        });
      });

      await waitFor(() => {
        expect(result.current.getItem(testArc.key)?.name).toEqual("updated-name");
      });
    });

    it("should remove arc from list when deleted", async () => {
      const testArc = await client.arcs.create({
        name: "to-delete",
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });

      const { result } = renderHook(() => Arc.useList({}), { wrapper });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data).toContain(testArc.key);

      await act(async () => {
        await client.arcs.delete(testArc.key);
      });

      await waitFor(() => {
        expect(result.current.data).not.toContain(testArc.key);
      });
    });

    it("should filter arcs by keys", async () => {
      const arc1 = await client.arcs.create({
        name: "filter-arc-1",
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });
      const arc2 = await client.arcs.create({
        name: "filter-arc-2",
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });
      await client.arcs.create({
        name: "filter-arc-3",
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });

      const { result } = renderHook(() => Arc.useList({}), { wrapper });

      act(() => {
        result.current.retrieve({ keys: [arc1.key, arc2.key] });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data).toHaveLength(2);
      expect(result.current.data).toContain(arc1.key);
      expect(result.current.data).toContain(arc2.key);

      const retrievedArc1 = result.current.getItem(arc1.key);
      expect(retrievedArc1?.name).toBe("filter-arc-1");
      const retrievedArc2 = result.current.getItem(arc2.key);
      expect(retrievedArc2?.name).toBe("filter-arc-2");
    });
  });

  describe("useDelete", () => {
    it("should delete a single arc", async () => {
      const testArc = await client.arcs.create({
        name: "delete-single",
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });

      const { result } = renderHook(() => Arc.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(testArc.key);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.arcs.retrieve({ key: testArc.key })).rejects.toThrow();
    });

    it("should delete multiple arcs", async () => {
      const arc1 = await client.arcs.create({
        name: "delete-multi-1",
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });
      const arc2 = await client.arcs.create({
        name: "delete-multi-2",
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });

      const { result } = renderHook(() => Arc.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([arc1.key, arc2.key]);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.arcs.retrieve({ key: arc1.key })).rejects.toThrow();
      await expect(client.arcs.retrieve({ key: arc2.key })).rejects.toThrow();
    });
  });

  describe("useCreate", () => {
    it("should create a new arc", async () => {
      const { result } = renderHook(() => Arc.useCreate(), { wrapper });

      const uniqueName = `created-arc-${Math.random().toString(36).substring(7)}`;

      await act(async () => {
        await result.current.updateAsync({
          name: uniqueName,
          version: "1.0.0",
          graph: { nodes: [], edges: [] },
          text: { raw: "" },
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
    });
  });

  describe("useForm", () => {
    it("should initialize with default values for new arc", async () => {
      const { result } = renderHook(() => Arc.useForm({ query: {} }), { wrapper });

      await waitFor(() => expect(result.current.variant).toBe("success"));

      const formData = result.current.form.value();
      expect(formData.name).toBe("");
      expect(formData.version).toBe("0.0.0");
      expect(formData.graph).toEqual({ nodes: [], edges: [] });
      expect(formData.text).toEqual({ raw: "" });
    });

    it("should create a new arc on save", async () => {
      const { result } = renderHook(() => Arc.useForm({ query: {} }), { wrapper });

      await waitFor(() => expect(result.current.variant).toBe("success"));

      const uniqueName = `form-arc-${Math.random().toString(36).substring(7)}`;

      act(() => {
        result.current.form.set("name", uniqueName);
        result.current.form.set("version", "2.0.0");
      });

      await act(async () => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toBe("success");
        expect(result.current.form.value().name).toEqual(uniqueName);
        expect(result.current.form.value().version).toEqual("2.0.0");
        expect(result.current.form.value().key).toBeDefined();
      });
    });

    it("should retrieve and edit existing arc", async () => {
      const existingArc = await client.arcs.create({
        name: `existing-arc-${Math.random().toString(36).substring(7)}`,
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });

      const { result } = renderHook(
        () => Arc.useForm({ query: { key: existingArc.key } }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toBe("success"));

      expect(result.current.form.value().name).toEqual(existingArc.name);
      expect(result.current.form.value().version).toEqual("1.0.0");

      act(() => {
        result.current.form.set("name", "edited-arc");
        result.current.form.set("version", "1.5.0");
      });

      await act(async () => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toBe("success");
        expect(result.current.form.value().name).toEqual("edited-arc");
        expect(result.current.form.value().version).toEqual("1.5.0");
      });

      const retrieved = await client.arcs.retrieve({ key: existingArc.key });
      expect(retrieved.name).toBe("edited-arc");
      expect(retrieved.version).toBe("1.5.0");
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a single arc", async () => {
      const testArc = await client.arcs.create({
        name: `retrieve-arc-${Math.random().toString(36).substring(7)}`,
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });

      const { result } = renderHook(() => Arc.useRetrieve({ key: testArc.key }), {
        wrapper,
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data?.key).toBe(testArc.key);
      expect(result.current.data?.name).toBe(testArc.name);
      expect(result.current.data?.version).toBe("1.0.0");
    });
  });

  describe("useRename", () => {
    it("should rename an arc", async () => {
      const testArc = await client.arcs.create({
        name: `original-${Math.random().toString(36).substring(7)}`,
        version: "1.0.0",
        graph: { nodes: [], edges: [] },
        text: { raw: "" },
      });

      const { result } = renderHook(() => Arc.useRename(), { wrapper });

      const newName = `renamed-${Math.random().toString(36).substring(7)}`;

      await act(async () => {
        await result.current.updateAsync({ key: testArc.key, name: newName });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.arcs.retrieve({ key: testArc.key });
      expect(retrieved.name).toBe(newName);
    });
  });
});
