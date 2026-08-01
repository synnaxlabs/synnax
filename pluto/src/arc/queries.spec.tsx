// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, status, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor, within } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Arc } from "@/arc";
import { Errors } from "@/errors";
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

  const createTestArc = async (overrides: Partial<arc.New> = {}): Promise<arc.Arc> =>
    await client.arcs.create({
      name: `arc_${id.create()}`,
      mode: "graph",
      graph: {
        nodes: [
          { key: "n1", position: { x: 0, y: 0 } },
          { key: "n2", position: { x: 10, y: 10 } },
        ],
        edges: [
          {
            key: "edge1",
            source: { node: "n1", param: "out" },
            target: { node: "n2", param: "in" },
            kind: arc.ir.EdgeKind.continuous,
          },
        ],
        inputs: {
          n1: { type: "constant", value: 0 },
          n2: { type: "constant", value: 1 },
        },
      },
      ...overrides,
    });

  // Populates the client cache with the arc at `key` via the suspending
  // useEnsureRetrieved. A single-hook bootstrap component keeps the suspending
  // hook from being followed by additional hooks, which trips a React 19
  // concurrent-replay warning.
  const loadArc = async (
    key: string,
    Wrapper: FC<PropsWithChildren> = wrapper,
  ): Promise<void> => {
    const Bootstrap = (): ReactElement => {
      Arc.useEnsureRetrieved({ key });
      return <div data-testid="loaded" />;
    };
    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary loading={null}>
            <Bootstrap />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });
    await within(utils.container).findByTestId("loaded");
  };

  const createAndLoadArc = async (
    overrides: Partial<arc.New> = {},
  ): Promise<arc.Arc> => {
    const a = await createTestArc(overrides);
    await loadArc(a.key);
    return a;
  };

  describe("useList", () => {
    it("should return a list of arcs", async () => {
      const arc1 = await client.arcs.create({
        name: "arc1",
        mode: "text",
      });
      const arc2 = await client.arcs.create({
        name: "arc2",
        mode: "text",
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
      const { result } = renderHook(
        () => ({ list: Arc.useList({}), create: Arc.useCreate() }),
        { wrapper },
      );

      act(() => {
        result.current.list.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.list.variant).toEqual("success");
      });

      const initialLength = result.current.list.data.length;

      await act(async () => {
        await result.current.create.updateAsync({
          name: "new-arc",
          mode: "text",
        });
      });

      await waitFor(() => {
        expect(result.current.list.data.length).toBe(initialLength + 1);
      });

      const newArc = result.current.list.data
        .map((key) => result.current.list.getItem(key))
        .find((arc) => arc?.name === "new-arc");
      expect(newArc).toBeDefined();
      expect(newArc?.name).toBe("new-arc");
    });

    it("should update when an arc is modified", async () => {
      const key = uuid.create();
      const { result } = renderHook(
        () => ({ list: Arc.useList({}), create: Arc.useCreate() }),
        { wrapper },
      );

      await act(async () => {
        await result.current.create.updateAsync({
          key,
          name: "original-name",
          mode: "text",
        });
      });

      act(() => {
        result.current.list.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.list.variant).toEqual("success");
      });

      expect(result.current.list.getItem(key)?.name).toEqual("original-name");

      await act(async () => {
        await client.arcs.dispatch(key, [arc.rename({ name: "updated-name" })]);
      });

      await waitFor(() => {
        expect(result.current.list.getItem(key)?.name).toEqual("updated-name");
      });
    });

    it("should remove arc from list when deleted", async () => {
      const testArc = await client.arcs.create({
        name: "to-delete",
        mode: "text",
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
      });
      const arc2 = await client.arcs.create({
        name: "filter-arc-2",
        mode: "text",
      });
      await client.arcs.create({
        name: "filter-arc-3",
        mode: "text",
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

    it("hydrates the cache for a list-only arc so its doc selectors resolve", async () => {
      // Created on a second client so the local session never wrote it; the
      // doc must arrive through the list fetch (or the change stream).
      const remote = createTestClient();
      const a = await remote.arcs.create({
        name: `hydrate-${id.create()}`,
        mode: "text",
      });
      const { result } = renderHook(
        () => ({
          list: Arc.useList({}),
          hasText: Arc.useSelectHasText({ key: a.key }),
        }),
        { wrapper },
      );

      act(() => {
        result.current.list.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.list.variant).toEqual("success");
        expect(result.current.hasText).toBe(true);
      });
    });

    it("reflects a live rename on an arc only loaded via the list", async () => {
      const originalName = `list-only-${id.create()}`;
      const a = await client.arcs.create({ name: originalName, mode: "text" });
      const { result } = renderHook(
        () => ({ list: Arc.useList({}), rename: Arc.useRename() }),
        { wrapper },
      );

      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.list.variant).toEqual("success");
        expect(result.current.list.getItem(a.key)?.name).toEqual(originalName);
      });

      const newName = `list-only-renamed-${id.create()}`;
      await act(async () => {
        await result.current.rename.updateAsync({ key: a.key, name: newName });
      });

      await waitFor(() => {
        expect(result.current.list.getItem(a.key)?.name).toEqual(newName);
      });
    });

    it("does not replace a loaded arc's cached entry when the list refetches", async () => {
      const a = await createAndLoadArc();
      const { result } = renderHook(
        () => ({ list: Arc.useList({}), nodes: Arc.useSelectAllNodes({ key: a.key }) }),
        { wrapper },
      );
      const initialNodes = result.current.nodes;
      expect(initialNodes.map((n) => n.key)).toEqual(["n1", "n2"]);

      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.list.variant).toEqual("success");
        expect(result.current.list.getItem(a.key)).toBeDefined();
      });

      expect(result.current.nodes).toBe(initialNodes);
    });
  });

  describe("useDelete", () => {
    it("should delete a single arc", async () => {
      const testArc = await client.arcs.create({
        name: "delete-single",
        mode: "text",
      });

      const { result } = renderHook(() => Arc.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(testArc.key);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.arcs.retrieve(testArc.key)).rejects.toThrow();
    });

    it("should delete multiple arcs", async () => {
      const arc1 = await client.arcs.create({
        name: "delete-multi-1",
        mode: "text",
      });
      const arc2 = await client.arcs.create({
        name: "delete-multi-2",
        mode: "text",
      });

      const { result } = renderHook(() => Arc.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([arc1.key, arc2.key]);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.arcs.retrieve(arc1.key)).rejects.toThrow();
      await expect(client.arcs.retrieve(arc2.key)).rejects.toThrow();
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
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
    });

    describe("without rack", () => {
      it("should create arc without task when no rack is specified", async () => {
        const { result } = renderHook(() => Arc.useCreate(), { wrapper });
        const name = `arc_no_rack_${id.create()}`;
        await act(async () => {
          await result.current.updateAsync({
            name,
            mode: "text",
          });
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("success");
        });
        const createdArc = await client.arcs.retrieve({ name });
        const children = await client.ontology.children.retrieve({
          ids: arc.ontologyID(createdArc.key),
        });
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
            rack: testRack.key,
          });
        });

        await waitFor(async () => {
          expect(result.current.variant).toEqual("success");
          const createdArc = await client.arcs.retrieve(key);
          const children = await client.ontology.children.retrieve({
            ids: arc.ontologyID(createdArc.key),
            types: ["task"],
          });
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
            rack: testRack.key,
          });
        });

        await waitFor(async () => {
          expect(result.current.variant).toEqual("success");
          const createdArc = await client.arcs.retrieve(key);
          const children = await client.ontology.children.retrieve({
            ids: arc.ontologyID(createdArc.key),
            types: ["task"],
          });
          expect(children).toHaveLength(1);
          const taskKey = children[0].id.key;
          const retrievedTask = await client.tasks.retrieve(taskKey);
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
            const children = await client.ontology.children.retrieve({
              ids: arc.ontologyID(existingArc.key),
            });
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
              rack: testRack.key,
            });
          });

          let originalTaskKey: task.Key = "";
          await waitFor(async () => {
            expect(createResult.current.variant).toEqual("success");
            const createdArc = await client.arcs.retrieve(arcKey);
            const childrenBefore = await client.ontology.children.retrieve({
              ids: arc.ontologyID(createdArc.key),
              types: ["task"],
            });
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
              rack: testRack.key,
            });
          });

          await waitFor(async () => {
            expect(updateResult.current.variant).toEqual("success");
            const childrenAfter = await client.ontology.children.retrieve({
              ids: arc.ontologyID(arcKey),
              types: ["task"],
            });
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
              rack: rack1.key,
            });
          });

          let originalTaskKey: task.Key = "";
          await waitFor(async () => {
            expect(createResult.current.variant).toEqual("success");
            const childrenBefore = await client.ontology.children.retrieve({
              ids: arc.ontologyID(arcKey),
              types: ["task"],
            });
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
              rack: rack2.key,
            });
          });

          await waitFor(async () => {
            expect(updateResult.current.variant).toEqual("success");
            const childrenAfter = await client.ontology.children.retrieve({
              ids: arc.ontologyID(arcKey),
              types: ["task"],
            });
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
              rack: rack1.key,
            });
          });

          let originalTaskKey: task.Key = "";
          await waitFor(async () => {
            expect(createResult.current.variant).toEqual("success");
            const childrenBefore = await client.ontology.children.retrieve({
              ids: arc.ontologyID(arcKey),
              types: ["task"],
            });
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
              rack: rack2.key,
            });
          });

          await waitFor(() => {
            expect(updateResult.current.variant).toEqual("success");
          });

          await expect(client.tasks.retrieve(originalTaskKey)).rejects.toThrow();
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
        inputs: {},
        functions: [],
      });
      expect(formData.text).toEqual({
        raw: "",
        doc: { inserts: [], deletes: [] },
      });
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

      const retrieved = await client.arcs.retrieve(existingArc.key);
      expect(retrieved.name).toBe("edited-arc");
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a single arc", async () => {
      const testArc = await client.arcs.create({
        name: `retrieve-arc-${Math.random().toString(36).substring(7)}`,
        mode: "text",
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

    it("does not suspend when the arc is already in the store", async () => {
      const a = await createTestArc();
      await loadArc(a.key);

      const Wrapper = wrapper;
      // With the store warm, the fast-path resolves without suspending.
      const Probe = (): ReactElement => {
        Arc.useEnsureRetrieved({ key: a.key });
        return <div data-testid="ready" />;
      };
      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary loading={<div data-testid="fallback" />}>
              <Probe />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });
      expect(utils.queryByTestId("fallback")).toBeNull();
      expect(utils.queryByTestId("ready")).toBeTruthy();
    });
  });

  describe("useRename", () => {
    it("should rename an arc", async () => {
      const testArc = await client.arcs.create({
        name: `original-${Math.random().toString(36).substring(7)}`,
        mode: "text",
      });

      const { result } = renderHook(() => Arc.useRename(), { wrapper });

      const newName = `renamed-${Math.random().toString(36).substring(7)}`;

      await act(async () => {
        await result.current.updateAsync({ key: testArc.key, name: newName });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.arcs.retrieve(testArc.key);
      expect(retrieved.name).toBe(newName);
    });
  });

  describe("useRetrieveTask", () => {
    it("should return null when no task is associated with arc", async () => {
      const testArc = await client.arcs.create({
        name: `arc-no-task-${Math.random().toString(36).substring(7)}`,
        mode: "text",
      });

      const { result } = renderHook(
        () => Arc.useRetrieveTask({ arcKey: testArc.key }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data).toBeNull();
    });

    it("should retrieve task associated with arc", async () => {
      const testArc = await client.arcs.create({
        name: `arc-with-task-${Math.random().toString(36).substring(7)}`,
        mode: "text",
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
      });

      const { result } = renderHook(
        () => Arc.useRetrieveTask({ arcKey: testArc.key }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data).toBeNull();

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
          cmd: "",
          data: {},
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

  describe("selectors", () => {
    let arcKey: arc.Key;
    beforeAll(async () => {
      arcKey = (await createAndLoadArc()).key;
    });

    it("useSelectAllNodes returns the graph nodes", () => {
      const { result } = renderHook(() => Arc.useSelectAllNodes({ key: arcKey }), {
        wrapper,
      });
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });

    it("useSelectNodes returns only the nodes matching the given keys", () => {
      const { result } = renderHook(
        () => Arc.useSelectNodes({ key: arcKey, keys: ["n1"] }),
        { wrapper },
      );
      expect(result.current.map((n) => n.key)).toEqual(["n1"]);
    });

    it("useSelectNodes returns an empty array when no keys are given", () => {
      const { result } = renderHook(
        () => Arc.useSelectNodes({ key: arcKey, keys: [] }),
        { wrapper },
      );
      expect(result.current).toEqual([]);
    });

    it("useSelectAllEdges returns the keyed diagram edges", () => {
      const { result } = renderHook(() => Arc.useSelectAllEdges({ key: arcKey }), {
        wrapper,
      });
      expect(result.current).toHaveLength(1);
      const edge = result.current[0];
      expect(edge.key).toBe("edge1");
      expect(edge.source).toEqual({ node: "n1", param: "out" });
      expect(edge.target).toEqual({ node: "n2", param: "in" });
    });

    it("useSelectNodeConfig returns the config for a node", () => {
      const { result } = renderHook(
        () => Arc.useSelectNodeConfig({ key: arcKey, nodeKey: "n1" }),
        { wrapper },
      );
      expect(result.current).toEqual({ type: "constant", value: 0 });
    });

    it("useSelectMode returns the representation mode", () => {
      const { result } = renderHook(() => Arc.useSelectMode({ key: arcKey }), {
        wrapper,
      });
      expect(result.current).toBe("graph");
    });

    it("useSelectMode reflects a text-mode arc", async () => {
      const { key } = await createAndLoadArc({ mode: "text" });
      const { result } = renderHook(() => Arc.useSelectMode({ key }), { wrapper });
      expect(result.current).toBe("text");
    });

    it("useSelectName returns the arc's name", async () => {
      const { key, name } = await createAndLoadArc();
      const { result } = renderHook(() => Arc.useSelectName({ key }), { wrapper });
      expect(result.current).toBe(name);
    });
  });

  describe("selector memoization & stability", () => {
    it("useSelectAllNodes keeps its reference when an unrelated change occurs", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          nodes: Arc.useSelectAllNodes({ key: isolated.key }),
          dispatch: Arc.useDispatch(),
        }),
        { wrapper },
      );
      const initial = result.current.nodes;
      expect(initial.map((n) => n.key)).toEqual(["n1", "n2"]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodeInputs({ key: "n1", inputs: { value: 99 } })],
        });
      });
      expect(result.current.nodes).toBe(initial);
    });

    it("useSelectNodes keeps its reference when an unselected node moves", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          nodes: Arc.useSelectNodes({ key: isolated.key, keys: ["n1"] }),
          dispatch: Arc.useDispatch(),
        }),
        { wrapper },
      );
      const initial = result.current.nodes;
      expect(initial.map((n) => n.key)).toEqual(["n1"]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n2", position: { x: 50, y: 50 } })],
        });
      });
      expect(result.current.nodes).toBe(initial);
    });

    it("useSelectAllNodes returns a new array when a node moves", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          nodes: Arc.useSelectAllNodes({ key: isolated.key }),
          dispatch: Arc.useDispatch(),
        }),
        { wrapper },
      );
      const initial = result.current.nodes;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n1", position: { x: 99, y: 99 } })],
        });
      });
      await waitFor(() => {
        expect(result.current.nodes).not.toBe(initial);
        expect(result.current.nodes.find((n) => n.key === "n1")?.position).toEqual({
          x: 99,
          y: 99,
        });
      });
    });

    it("useSelectAllEdges keeps its transformed reference when a node moves", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          edges: Arc.useSelectAllEdges({ key: isolated.key }),
          dispatch: Arc.useDispatch(),
        }),
        { wrapper },
      );
      const initial = result.current.edges;
      expect(initial).toHaveLength(1);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n1", position: { x: 7, y: 7 } })],
        });
      });
      expect(result.current.edges).toBe(initial);
    });

    it("useSelectAllEdges returns a new array when an edge is added", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          edges: Arc.useSelectAllEdges({ key: isolated.key }),
          dispatch: Arc.useDispatch(),
        }),
        { wrapper },
      );
      const initial = result.current.edges;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [
            arc.addEdge({
              edge: {
                key: "edge2",
                source: { node: "n2", param: "out" },
                target: { node: "n1", param: "in" },
                kind: arc.ir.EdgeKind.continuous,
              },
            }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.edges).not.toBe(initial);
        expect(result.current.edges).toHaveLength(2);
      });
    });

    it("useSelectNodeConfig keeps its reference when a different node's config changes", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          config: Arc.useSelectNodeConfig({ key: isolated.key, nodeKey: "n1" }),
          dispatch: Arc.useDispatch(),
        }),
        { wrapper },
      );
      const initial = result.current.config;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodeInputs({ key: "n2", inputs: { value: 42 } })],
        });
      });
      expect(result.current.config).toBe(initial);
    });

    it("useSelectNodeConfig returns a new value when the node's own config changes", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          config: Arc.useSelectNodeConfig({ key: isolated.key, nodeKey: "n1" }),
          dispatch: Arc.useDispatch(),
        }),
        { wrapper },
      );
      const initial = result.current.config;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodeInputs({ key: "n1", inputs: { value: 42 } })],
        });
      });
      await waitFor(() => {
        expect(result.current.config).not.toBe(initial);
        expect(result.current.config).toEqual({ type: "constant", value: 42 });
      });
    });

    it("useSelectName keeps its value when an unrelated change occurs", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          name: Arc.useSelectName({ key: isolated.key }),
          dispatch: Arc.useDispatch(),
        }),
        { wrapper },
      );
      const initial = result.current.name;
      expect(initial).toBe(isolated.name);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n1", position: { x: 13, y: 13 } })],
        });
      });
      expect(result.current.name).toBe(initial);
    });

    it("useSelectName reflects a live rename", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          name: Arc.useSelectName({ key: isolated.key }),
          rename: Arc.useRename(),
        }),
        { wrapper },
      );
      expect(result.current.name).toBe(isolated.name);
      const newName = `renamed_${id.create()}`;
      await act(async () => {
        await result.current.rename.updateAsync({ key: isolated.key, name: newName });
      });
      await waitFor(() => expect(result.current.name).toBe(newName));
    });
  });

  describe("useDispatch", () => {
    it("applies an action and updates the cache", async () => {
      const isolated = await createAndLoadArc();
      const { result: nodes } = renderHook(
        () => Arc.useSelectAllNodes({ key: isolated.key }),
        { wrapper },
      );
      expect(nodes.current.find((n) => n.key === "n1")?.position).toEqual({
        x: 0,
        y: 0,
      });
      const { result: dispatchHook } = renderHook(() => Arc.useDispatch(), {
        wrapper,
      });
      await act(async () => {
        await dispatchHook.current.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n1", position: { x: 100, y: 200 } })],
        });
      });
      await waitFor(() =>
        expect(nodes.current.find((n) => n.key === "n1")?.position).toEqual({
          x: 100,
          y: 200,
        }),
      );
    });

    it("applies multiple actions in a single dispatch", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          nodes: Arc.useSelectAllNodes({ key: isolated.key }),
          config: Arc.useSelectNodeConfig({ key: isolated.key, nodeKey: "n3" }),
          dispatch: Arc.useDispatch(),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [
            arc.setNode({ node: { key: "n3", position: { x: 5, y: 5 } } }),
            arc.setNodeInputs({ key: "n3", inputs: { type: "constant", value: 7 } }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.nodes.find((n) => n.key === "n3")?.position).toEqual({
          x: 5,
          y: 5,
        });
        expect(result.current.config).toEqual({ type: "constant", value: 7 });
      });
    });

    it("propagates a dispatched action to a second wrapper", async () => {
      const isolated = await createAndLoadArc();
      const wrapperB = await createAsyncSynnaxWrapper({ client });
      await loadArc(isolated.key, wrapperB);

      const { result: nodesB } = renderHook(
        () => Arc.useSelectAllNodes({ key: isolated.key }),
        { wrapper: wrapperB },
      );
      const { result: dispatchHook } = renderHook(() => Arc.useDispatch(), {
        wrapper,
      });

      await act(async () => {
        await dispatchHook.current.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n1", position: { x: 50, y: 50 } })],
        });
      });

      await waitFor(() =>
        expect(nodesB.current.find((n) => n.key === "n1")?.position).toEqual({
          x: 50,
          y: 50,
        }),
      );
    });
  });

  describe("useUndo / useRedo", () => {
    it("undo reverts a dispatched action", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          nodes: Arc.useSelectAllNodes({ key: isolated.key }),
          dispatch: Arc.useDispatch(),
          undo: Arc.useUndo({ key: isolated.key }),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n1", position: { x: 80, y: 80 } })],
        });
      });
      await waitFor(() =>
        expect(result.current.nodes.find((n) => n.key === "n1")?.position).toEqual({
          x: 80,
          y: 80,
        }),
      );
      await act(async () => {
        result.current.undo.undo();
      });
      await waitFor(() =>
        expect(result.current.nodes.find((n) => n.key === "n1")?.position).toEqual({
          x: 0,
          y: 0,
        }),
      );
    });

    it("coalesces consecutive moves into a single undo step", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          nodes: Arc.useSelectAllNodes({ key: isolated.key }),
          dispatch: Arc.useDispatch(),
          undo: Arc.useUndo({ key: isolated.key }),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n1", position: { x: 20, y: 0 } })],
        });
      });
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n1", position: { x: 40, y: 0 } })],
        });
      });
      await waitFor(() =>
        expect(result.current.nodes.find((n) => n.key === "n1")?.position).toEqual({
          x: 40,
          y: 0,
        }),
      );
      await act(async () => {
        result.current.undo.undo();
      });
      await waitFor(() =>
        expect(result.current.nodes.find((n) => n.key === "n1")?.position).toEqual({
          x: 0,
          y: 0,
        }),
      );
    });

    it("redo reapplies an undone action", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          nodes: Arc.useSelectAllNodes({ key: isolated.key }),
          dispatch: Arc.useDispatch(),
          undo: Arc.useUndo({ key: isolated.key }),
          redo: Arc.useRedo({ key: isolated.key }),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n1", position: { x: 60, y: 60 } })],
        });
      });
      await act(async () => {
        result.current.undo.undo();
      });
      await waitFor(() =>
        expect(result.current.nodes.find((n) => n.key === "n1")?.position).toEqual({
          x: 0,
          y: 0,
        }),
      );
      await act(async () => {
        result.current.redo.redo();
      });
      await waitFor(() =>
        expect(result.current.nodes.find((n) => n.key === "n1")?.position).toEqual({
          x: 60,
          y: 60,
        }),
      );
    });

    it("exposes canUndo and canRedo availability", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          dispatch: Arc.useDispatch(),
          undo: Arc.useUndo({ key: isolated.key }),
          redo: Arc.useRedo({ key: isolated.key }),
        }),
        { wrapper },
      );
      expect(result.current.undo.canUndo).toBe(false);
      expect(result.current.redo.canRedo).toBe(false);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [arc.setNodePosition({ key: "n1", position: { x: 12, y: 34 } })],
        });
      });
      await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
      await act(async () => {
        result.current.undo.undo();
      });
      await waitFor(() => expect(result.current.redo.canRedo).toBe(true));
    });
  });

  describe("useAddNode", () => {
    it("appends a node and seeds its config from the registry", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          add: Arc.useAddNode(isolated.key),
          nodes: Arc.useSelectAllNodes({ key: isolated.key }),
          config: Arc.useSelectNodeConfig({ key: isolated.key, nodeKey: "added" }),
        }),
        { wrapper },
      );
      await act(async () => {
        result.current.add({
          key: "added",
          type: "constant",
          position: { x: 1, y: 2 },
        });
      });
      await waitFor(() => {
        expect(result.current.nodes.find((n) => n.key === "added")?.position).toEqual({
          x: 1,
          y: 2,
        });
        expect(result.current.config).toEqual({ type: "constant", value: 0 });
      });
    });

    it("ignores unknown function types", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          add: Arc.useAddNode(isolated.key),
          nodes: Arc.useSelectAllNodes({ key: isolated.key }),
        }),
        { wrapper },
      );
      const initialLength = result.current.nodes.length;
      await act(async () => {
        result.current.add({ key: "ghost", type: "does_not_exist" });
      });
      expect(result.current.nodes.find((n) => n.key === "ghost")).toBeUndefined();
      expect(result.current.nodes).toHaveLength(initialLength);
    });
  });

  describe("useEnsureRetrieved", () => {
    it("populates the cache so selectors resolve", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => Arc.useSelectAllNodes({ key: isolated.key }),
        {
          wrapper,
        },
      );
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });
  });

  describe("useRetrieveObservable", () => {
    it("fires onChange with the arc initially and on each change", async () => {
      const isolated = await createTestArc();
      const seen: string[] = [];
      const { result } = renderHook(
        () => ({
          obs: Arc.useRetrieveObservable({
            onChange: (res) => {
              if (res.variant === "success") seen.push(res.data.name);
            },
          }),
          rename: Arc.useRename(),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.obs.retrieveAsync({ key: isolated.key });
      });
      await waitFor(() => expect(seen).toContain(isolated.name));
      await act(async () => {
        await result.current.rename.updateAsync({
          key: isolated.key,
          name: "obs_renamed",
        });
      });
      await waitFor(() => expect(seen).toContain("obs_renamed"));
    });
  });
});
