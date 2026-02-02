// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, createTestClient, task } from "@synnaxlabs/client";
import { id, status, uuid } from "@synnaxlabs/x";
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
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
        text: { raw: "" },
      });
      const arc2 = await client.arcs.create({
        name: "arc2",
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
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
          mode: "text",
          graph: {
            nodes: [],
            edges: [],
            viewport: { position: { x: 0, y: 0 }, zoom: 1 },
            functions: [],
          },
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
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
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
          mode: "text",
        });
      });

      await waitFor(() => {
        expect(result.current.getItem(testArc.key)?.name).toEqual("updated-name");
      });
    });

    it("should remove arc from list when deleted", async () => {
      const testArc = await client.arcs.create({
        name: "to-delete",
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
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
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
        text: { raw: "" },
      });
      const arc2 = await client.arcs.create({
        name: "filter-arc-2",
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
        text: { raw: "" },
      });
      await client.arcs.create({
        name: "filter-arc-3",
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
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
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
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
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
        text: { raw: "" },
      });
      const arc2 = await client.arcs.create({
        name: "delete-multi-2",
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
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
          mode: "text",
          graph: {
            nodes: [],
            edges: [],
            viewport: { position: { x: 0, y: 0 }, zoom: 1 },
            functions: [],
          },
          text: { raw: "" },
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
    });

    describe("without rack", () => {
      it("should create arc without task when no rack is specified", async () => {
        const { result } = renderHook(() => Arc.useCreate(), { wrapper });
        await act(async () => {
          await result.current.updateAsync({
            name: `arc_no_rack_${id.create()}`,
            mode: "text",
            graph: {
              nodes: [],
              edges: [],
              viewport: { position: { x: 0, y: 0 }, zoom: 1 },
              functions: [],
            },
            text: { raw: "" },
          });
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("success");
        });
        expect(result.current.data).toBeDefined();
        const createdArc = result.current.data!;
        const children = await client.ontology.retrieveChildren(
          arc.ontologyID(createdArc.key as arc.Key),
        );
        const taskChildren = children.filter((c) => c.id.type === "task");
        expect(taskChildren).toHaveLength(0);
      });
    });

    describe("with rack", () => {
      it("should create arc with new task when rack is specified", async () => {
        const testRack = await client.racks.create({ name: `rack_new_${id.create()}` });
        const { result } = renderHook(() => Arc.useCreate(), { wrapper });

        const key = uuid.create();

        await act(async () => {
          await result.current.updateAsync({
            key,
            name: `arc_with_rack_${id.create()}`,
            mode: "text",
            graph: {
              nodes: [],
              edges: [],
              viewport: { position: { x: 0, y: 0 }, zoom: 1 },
              functions: [],
            },
            text: { raw: "" },
            rack: testRack.key,
          });
        });

        await waitFor(async () => {
          expect(result.current.variant).toEqual("success");
          const createdArc = await client.arcs.retrieve({ key });
          const children = await client.ontology.retrieveChildren(
            arc.ontologyID(createdArc.key),
            { types: ["task"] },
          );
          expect(children).toHaveLength(1);
        });
      });

      it("should set task type to arc and config with arcKey", async () => {
        const testRack = await client.racks.create({
          name: `rack_config_${id.create()}`,
        });
        const { result } = renderHook(() => Arc.useCreate(), { wrapper });

        const key = uuid.create();

        await act(async () => {
          await result.current.updateAsync({
            key,
            name: `arc_config_${id.create()}`,
            mode: "text",
            graph: {
              nodes: [],
              edges: [],
              viewport: { position: { x: 0, y: 0 }, zoom: 1 },
              functions: [],
            },
            text: { raw: "" },
            rack: testRack.key,
          });
        });

        await waitFor(async () => {
          expect(result.current.variant).toEqual("success");
          const createdArc = await client.arcs.retrieve({ key });
          const children = await client.ontology.retrieveChildren(
            arc.ontologyID(createdArc.key),
            { types: ["task"] },
          );
          expect(children).toHaveLength(1);
          const taskKey = children[0].id.key;
          const retrievedTask = await client.tasks.retrieve({ key: taskKey });
          expect(retrievedTask.type).toBe("arc");
          expect(retrievedTask.config).toEqual({ arcKey: createdArc.key });
          expect(task.rackKey(taskKey)).toBe(testRack.key);
        });
      });

      describe("existing arc", () => {
        it("should create new task when updating arc that has no task", async () => {
          const existingArc = await client.arcs.create({
            name: `existing_no_task_${id.create()}`,
            mode: "text",
            graph: {
              nodes: [],
              edges: [],
              viewport: { position: { x: 0, y: 0 }, zoom: 1 },
              functions: [],
            },
            text: { raw: "" },
          });

          const testRack = await client.racks.create({
            name: `rack_update_${id.create()}`,
          });
          const { result } = renderHook(() => Arc.useCreate(), { wrapper });

          await act(async () => {
            await result.current.updateAsync({
              key: existingArc.key,
              mode: existingArc.mode,
              name: existingArc.name,
              graph: existingArc.graph,
              text: existingArc.text,
              rack: testRack.key,
            });
          });

          await waitFor(async () => {
            expect(result.current.variant).toEqual("success");
            const children = await client.ontology.retrieveChildren(
              arc.ontologyID(existingArc.key),
            );
            const taskChildren = children.filter((c) => c.id.type === "task");
            expect(taskChildren).toHaveLength(1);
            expect(task.rackKey(taskChildren[0].id.key)).toBe(testRack.key);
          });
        });

        it("should reuse task key when updating arc on same rack", async () => {
          const testRack = await client.racks.create({
            name: `rack_reuse_${id.create()}`,
          });
          const { result: createResult } = renderHook(() => Arc.useCreate(), {
            wrapper,
          });

          const arcKey = uuid.create();
          const uniqueName = `arc_reuse_${id.create()}`;

          await act(async () => {
            await createResult.current.updateAsync({
              key: arcKey,
              mode: "text",
              name: uniqueName,
              graph: {
                nodes: [],
                edges: [],
                viewport: { position: { x: 0, y: 0 }, zoom: 1 },
                functions: [],
              },
              text: { raw: "" },
              rack: testRack.key,
            });
          });

          let originalTaskKey: task.Key = "";
          await waitFor(async () => {
            expect(createResult.current.variant).toEqual("success");
            const createdArc = await client.arcs.retrieve({ key: arcKey });
            const childrenBefore = await client.ontology.retrieveChildren(
              arc.ontologyID(createdArc.key),
              { types: ["task"] },
            );
            expect(childrenBefore).toHaveLength(1);
            originalTaskKey = childrenBefore[0].id.key;
          });

          const { result: updateResult } = renderHook(() => Arc.useCreate(), {
            wrapper,
          });

          await act(async () => {
            await updateResult.current.updateAsync({
              key: arcKey,
              mode: "text",
              name: `${uniqueName}_updated`,
              graph: {
                nodes: [],
                edges: [],
                viewport: { position: { x: 0, y: 0 }, zoom: 1 },
                functions: [],
              },
              text: { raw: "" },
              rack: testRack.key,
            });
          });

          await waitFor(async () => {
            expect(updateResult.current.variant).toEqual("success");
            const childrenAfter = await client.ontology.retrieveChildren(
              arc.ontologyID(arcKey),
              { types: ["task"] },
            );
            expect(childrenAfter).toHaveLength(1);
            expect(childrenAfter[0].id.key).toBe(originalTaskKey);
          });
        });

        it("should migrate task when updating arc to different rack", async () => {
          const rack1 = await client.racks.create({
            name: `rack_from_${id.create()}`,
          });
          const rack2 = await client.racks.create({ name: `rack_to_${id.create()}` });
          const { result: createResult } = renderHook(() => Arc.useCreate(), {
            wrapper,
          });

          const arcKey = uuid.create();

          await act(async () => {
            await createResult.current.updateAsync({
              key: arcKey,
              mode: "text",
              name: `arc_migrate_${id.create()}`,
              graph: {
                nodes: [],
                edges: [],
                viewport: { position: { x: 0, y: 0 }, zoom: 1 },
                functions: [],
              },
              text: { raw: "" },
              rack: rack1.key,
            });
          });

          let originalTaskKey: task.Key = "" as task.Key;
          await waitFor(async () => {
            expect(createResult.current.variant).toEqual("success");
            const childrenBefore = await client.ontology.retrieveChildren(
              arc.ontologyID(arcKey),
              { types: ["task"] },
            );
            expect(childrenBefore).toHaveLength(1);
            originalTaskKey = childrenBefore[0].id.key;
            expect(task.rackKey(originalTaskKey)).toBe(rack1.key);
          });

          const { result: updateResult } = renderHook(() => Arc.useCreate(), {
            wrapper,
          });

          await act(async () => {
            await updateResult.current.updateAsync({
              key: arcKey,
              mode: "text",
              name: `arc_migrate_updated`,
              graph: {
                nodes: [],
                edges: [],
                viewport: { position: { x: 0, y: 0 }, zoom: 1 },
                functions: [],
              },
              text: { raw: "" },
              rack: rack2.key,
            });
          });

          await waitFor(async () => {
            expect(updateResult.current.variant).toEqual("success");
            const childrenAfter = await client.ontology.retrieveChildren(
              arc.ontologyID(arcKey),
              { types: ["task"] },
            );
            expect(childrenAfter).toHaveLength(1);
            const newTaskKey = childrenAfter[0].id.key;
            expect(newTaskKey).not.toBe(originalTaskKey);
            expect(task.rackKey(newTaskKey)).toBe(rack2.key);
          });
        });

        it("should delete old task when migrating to different rack", async () => {
          const rack1 = await client.racks.create({
            name: `rack_del_from_${id.create()}`,
          });
          const rack2 = await client.racks.create({
            name: `rack_del_to_${id.create()}`,
          });
          const { result: createResult } = renderHook(() => Arc.useCreate(), {
            wrapper,
          });

          const arcKey = uuid.create();

          await act(async () => {
            await createResult.current.updateAsync({
              key: arcKey,
              mode: "text",
              name: `arc_del_${id.create()}`,
              graph: {
                nodes: [],
                edges: [],
                viewport: { position: { x: 0, y: 0 }, zoom: 1 },
                functions: [],
              },
              text: { raw: "" },
              rack: rack1.key,
            });
          });

          let originalTaskKey: task.Key = "" as task.Key;
          await waitFor(async () => {
            expect(createResult.current.variant).toEqual("success");
            const childrenBefore = await client.ontology.retrieveChildren(
              arc.ontologyID(arcKey),
              { types: ["task"] },
            );
            expect(childrenBefore).toHaveLength(1);
            originalTaskKey = childrenBefore[0].id.key;
          });

          const { result: updateResult } = renderHook(() => Arc.useCreate(), {
            wrapper,
          });

          await act(async () => {
            await updateResult.current.updateAsync({
              key: arcKey,
              mode: "text",
              name: `arc_del_updated`,
              graph: {
                nodes: [],
                edges: [],
                viewport: { position: { x: 0, y: 0 }, zoom: 1 },
                functions: [],
              },
              text: { raw: "" },
              rack: rack2.key,
            });
          });

          await waitFor(() => {
            expect(updateResult.current.variant).toEqual("success");
          });

          await expect(
            client.tasks.retrieve({ key: originalTaskKey }),
          ).rejects.toThrow();
        });
      });
    });
  });

  describe("useForm", () => {
    it("should initialize with default values for new arc", async () => {
      const { result } = renderHook(() => Arc.useForm({ query: {} }), { wrapper });

      await waitFor(() => expect(result.current.variant).toBe("success"));

      const formData = result.current.form.value();
      expect(formData.name).toBe("");
      expect(formData.graph).toEqual({
        nodes: [],
        edges: [],
        viewport: { position: { x: 0, y: 0 }, zoom: 1 },
        functions: [],
      });
      expect(formData.text).toEqual({ raw: "" });
    });

    it("should create a new arc on save", async () => {
      const { result } = renderHook(() => Arc.useForm({ query: {} }), { wrapper });

      await waitFor(() => expect(result.current.variant).toBe("success"));

      const uniqueName = `form-arc-${Math.random().toString(36).substring(7)}`;

      act(() => {
        result.current.form.set("name", uniqueName);
      });

      await act(async () => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toBe("success");
        expect(result.current.form.value().name).toEqual(uniqueName);
        expect(result.current.form.value().key).toBeDefined();
      });
    });

    it("should retrieve and edit existing arc", async () => {
      const existingArc = await client.arcs.create({
        name: `existing-arc-${Math.random().toString(36).substring(7)}`,
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
        text: { raw: "" },
      });

      const { result } = renderHook(
        () => Arc.useForm({ query: { key: existingArc.key } }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toBe("success");
      });

      expect(result.current.form.value().name).toEqual(existingArc.name);

      act(() => {
        result.current.form.set("name", "edited-arc");
      });

      await act(async () => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toBe("success");
        expect(result.current.form.value().name).toEqual("edited-arc");
      });

      const retrieved = await client.arcs.retrieve({ key: existingArc.key });
      expect(retrieved.name).toBe("edited-arc");
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a single arc", async () => {
      const testArc = await client.arcs.create({
        name: `retrieve-arc-${Math.random().toString(36).substring(7)}`,
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
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
    });
  });

  describe("useRename", () => {
    it("should rename an arc", async () => {
      const testArc = await client.arcs.create({
        name: `original-${Math.random().toString(36).substring(7)}`,
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
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

  describe("useRetrieveTask", () => {
    it("should return undefined when no task is associated with arc", async () => {
      const testArc = await client.arcs.create({
        name: `arc-no-task-${Math.random().toString(36).substring(7)}`,
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
        text: { raw: "" },
      });

      const { result } = renderHook(
        () => Arc.useRetrieveTask({ arcKey: testArc.key }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data).toBeUndefined();
    });

    it("should retrieve task associated with arc", async () => {
      const testArc = await client.arcs.create({
        name: `arc-with-task-${Math.random().toString(36).substring(7)}`,
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
        text: { raw: "" },
      });

      const rack = await client.racks.create({ name: "test-rack" });
      const testTask = await rack.createTask({
        name: "arc-task",
        type: "testType",
        config: { value: "test" },
      });

      await client.ontology.addChildren(
        arc.ontologyID(testArc.key),
        task.ontologyID(testTask.key),
      );

      const { result } = renderHook(
        () => Arc.useRetrieveTask({ arcKey: testArc.key }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toBeDefined();
      });

      expect(result.current.data?.key).toEqual(testTask.key);
      expect(result.current.data?.name).toEqual("arc-task");
      expect(result.current.data?.config).toEqual({ value: "test" });
    });

    it("should update when a task is associated with arc", async () => {
      const testArc = await client.arcs.create({
        name: `arc-task-add-${Math.random().toString(36).substring(7)}`,
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
        text: { raw: "" },
      });

      const { result } = renderHook(
        () => Arc.useRetrieveTask({ arcKey: testArc.key }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data).toBeUndefined();

      const rack = await client.racks.create({ name: "test-rack-add" });
      const testTask = await rack.createTask({
        name: "new-arc-task",
        type: "testType",
        config: {},
      });

      await act(async () => {
        await client.ontology.addChildren(
          arc.ontologyID(testArc.key),
          task.ontologyID(testTask.key),
        );
      });

      await waitFor(() => {
        expect(result.current.data).toBeDefined();
        expect(result.current.data?.key).toEqual(testTask.key);
      });
    });

    it("should update when task status changes", async () => {
      const testArc = await client.arcs.create({
        name: `arc-status-${Math.random().toString(36).substring(7)}`,
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
        text: { raw: "" },
      });

      const rack = await client.racks.create({ name: "test-rack-status" });
      const testTask = await rack.createTask({
        name: "status-task",
        type: "testType",
        config: {},
      });

      await client.ontology.addChildren(
        arc.ontologyID(testArc.key),
        task.ontologyID(testTask.key),
      );

      const { result } = renderHook(
        () => Arc.useRetrieveTask({ arcKey: testArc.key }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toBeDefined();
      });

      const taskStatus: task.Status = status.create<
        ReturnType<typeof task.statusDetailsZ>
      >({
        key: task.statusKey(testTask.key),
        variant: "error",
        message: "Task failed",
        details: {
          task: testTask.key,
          running: false,
          data: undefined,
        },
      });

      await act(async () => {
        await client.statuses.set(taskStatus);
      });

      await waitFor(() => {
        expect(result.current.data?.status?.variant).toEqual("error");
        expect(result.current.data?.status?.message).toEqual("Task failed");
      });
    });

    it("should update when task is renamed", async () => {
      const testArc = await client.arcs.create({
        name: `arc-rename-${Math.random().toString(36).substring(7)}`,
        mode: "text",
        graph: {
          nodes: [],
          edges: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          functions: [],
        },
        text: { raw: "" },
      });

      const rack = await client.racks.create({ name: "test-rack-rename" });
      const testTask = await rack.createTask({
        name: "original-task-name",
        type: "testType",
        config: {},
      });

      await client.ontology.addChildren(
        arc.ontologyID(testArc.key),
        task.ontologyID(testTask.key),
      );

      const { result } = renderHook(
        () => Arc.useRetrieveTask({ arcKey: testArc.key }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data?.name).toEqual("original-task-name");
      });

      await act(async () => {
        await client.tasks.create({
          ...testTask.payload,
          name: "renamed-task-name",
        });
      });

      await waitFor(() => {
        expect(result.current.data?.name).toEqual("renamed-task-name");
      });
    });
  });
});
