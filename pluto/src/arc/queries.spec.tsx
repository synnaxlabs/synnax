// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, query, status, task } from "@synnaxlabs/client";
import {
  createTestClient,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { crdt, deep, id } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor, within } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import {
  afterEach,
  assert,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";

import { Arc } from "@/arc";
import { Errors } from "@/errors";
import { Flux } from "@/flux";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const path = Arc.formPath;

const graphOf = (a: arc.Arc): arc.graph.Graph => {
  if (a.source.mode !== "graph") throw new Error("expected a graph-mode Arc");
  return a.source.graph;
};

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
      type: "automation",
      source: {
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
      },
      ...overrides,
    });

  // Populates the client cache with the arc at `key` via the suspending useEnsure. A
  // single-hook bootstrap component keeps the suspending hook from being followed by
  // additional hooks, which trips a React 19 concurrent-replay warning.
  const loadArc = async (
    key: string,
    Wrapper: FC<PropsWithChildren> = wrapper,
  ): Promise<void> => {
    const Bootstrap = (): ReactElement => {
      Arc.useEnsure({ key });
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

  describe("list", () => {
    const named = (name: string) => ({ where: { name: { in: [name] } } });

    it("hydrates the cache for a list-only arc so its doc selectors resolve", async () => {
      // Created on a second client so the local session never wrote it; the
      // doc must arrive through the list fetch (or the change stream).
      const remote = createTestClient();
      const a = await remote.arcs.create({
        name: `hydrate-${id.create()}`,
        type: "automation",
        source: { mode: "text" },
      });
      const { result } = renderHook(
        () => Flux.useList((c) => c.arcs, { name: "Arcs", query: named(a.name) }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([a.key]));

      const { result: hasText } = renderHook(() => Arc.useHasText({ key: a.key }), {
        wrapper,
      });
      expect(hasText.current).toBe(true);
    });

    it("does not replace a loaded arc's cached entry when the list fetches", async () => {
      const a = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          list: Flux.useList((c) => c.arcs, { name: "Arcs", query: named(a.name) }),
          nodes: Arc.useAllNodes({ key: a.key }),
        }),
        { wrapper },
      );
      const initialNodes = result.current.nodes;
      expect(initialNodes.map((n) => n.key)).toEqual(["n1", "n2"]);
      act(() => result.current.list.more());
      await waitFor(() => expect(result.current.list.data).toEqual([a.key]));
      expect(result.current.nodes).toBe(initialNodes);
    });
  });

  describe("useResultList", () => {
    it("finds an arc by its name", async () => {
      const a = await createTestArc();
      const { result } = renderHook(() => Arc.useResultList({ searchTerm: a.name }), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data?.map(({ key }) => key)).toContain(a.key);
      });
    });
  });

  describe("useDelete", () => {
    it("should delete a single arc", async () => {
      const testArc = await client.arcs.create({
        name: "delete-single",
        type: "automation",
        source: { mode: "text" },
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
        type: "automation",
        source: { mode: "text" },
      });
      const arc2 = await client.arcs.create({
        name: "delete-multi-2",
        type: "automation",
        source: { mode: "text" },
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
          type: "automation",
          source: { mode: "text" },
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
    });

    it("does not attach a deployment task", async () => {
      const { result } = renderHook(() => Arc.useCreate(), { wrapper });
      const name = `arc_no_task_${id.create()}`;
      await act(async () => {
        await result.current.updateAsync({
          name,
          type: "automation",
          source: { mode: "text" },
        });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      const createdArc = await client.arcs.retrieve({ name });
      expect(await client.arcs.task.retrieve(createdArc.key)).toBeNull();
    });
  });

  describe("useUpdateTask", () => {
    it("creates the task stamped with the arc's hash", async () => {
      const testRack = await client.racks.create({ name: `rack_${id.create()}` });
      const created = await client.arcs.create({
        name: `arc_deploy_${id.create()}`,
        type: "automation",
        source: { mode: "text" },
      });
      const { result } = renderHook(() => Arc.useUpdateTask(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: created.key, rack: testRack.key });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const tsk = await client.arcs.task.retrieve(created.key);
      assert(tsk != null);
      expect(tsk.type).toBe("arc");
      expect(tsk.rack).toBe(testRack.key);
      expect(tsk.config.arcKey).toBe(created.key);
      expect(tsk.config.hash).not.toEqual("");
    });

    it("reuses the task across rebinds and rack moves", async () => {
      const rack1 = await client.racks.create({ name: `rack_${id.create()}` });
      const rack2 = await client.racks.create({ name: `rack_${id.create()}` });
      const created = await client.arcs.create({
        name: `arc_move_${id.create()}`,
        type: "automation",
        source: { mode: "text" },
      });
      const { result } = renderHook(() => Arc.useUpdateTask(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: created.key, rack: rack1.key });
      });
      const first = await client.arcs.task.retrieve(created.key);
      assert(first != null);
      expect(first.rack).toBe(rack1.key);

      await act(async () => {
        await result.current.updateAsync({ key: created.key, rack: rack2.key });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const moved = await client.tasks.retrieve(first.key);
      expect(moved.rack).toBe(rack2.key);
      const arcTasks = await client.tasks.retrieve({ arcs: [created.key] });
      expect(arcTasks.map((t) => t.key)).toEqual([first.key]);
    });

    it("clears the rack with a zero rack, deleting the task", async () => {
      const testRack = await client.racks.create({ name: `rack_${id.create()}` });
      const created = await client.arcs.create({
        name: `arc_undeploy_${id.create()}`,
        type: "automation",
        source: { mode: "text" },
      });
      const { result } = renderHook(() => Arc.useUpdateTask(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: created.key, rack: testRack.key });
      });
      const tsk = await client.arcs.task.retrieve(created.key);
      assert(tsk != null);

      await act(async () => {
        await result.current.updateAsync({ key: created.key, rack: 0 });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.tasks.retrieve(tsk.key)).rejects.toThrow();
      expect((await client.arcs.retrieve(created.key)).key).toEqual(created.key);
    });
    it("sets the performance of a bound arc", async () => {
      const testRack = await client.racks.create({ name: `rack_${id.create()}` });
      const created = await client.arcs.create({
        name: `arc_performance_${id.create()}`,
        type: "automation",
        source: { mode: "text" },
      });
      const { result } = renderHook(() => Arc.useUpdateTask(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: created.key, rack: testRack.key });
      });
      await act(async () => {
        await result.current.updateAsync({ key: created.key, performance: "medium" });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const tsk = await client.arcs.task.retrieve(created.key);
      assert(tsk != null);
      expect(tsk.rack).toBe(testRack.key);
      expect(tsk.config.performance).toBe("medium");
    });
  });

  describe("useForm", () => {
    it("should initialize with default values for a new arc", async () => {
      const { result } = renderHook(() => Arc.useForm({ query: {} }), { wrapper });

      await waitFor(() => expect(result.current.variant).toBe("success"));

      const formData = result.current.form.value();
      expect(formData.name).toBe("");
      expect(formData.mode).toBe("text");
      expect(formData.type).toBe("automation");
      expect(formData.key).toBeUndefined();
    });

    it("should create a new arc on save", async () => {
      const { result } = renderHook(() => Arc.useForm({ query: {} }), { wrapper });

      await waitFor(() => expect(result.current.variant).toBe("success"));

      const uniqueName = `form-arc-${Math.random().toString(36).substring(7)}`;

      act(() => {
        result.current.form.set(path.name, uniqueName);
        result.current.form.set(path.mode, "text");
      });

      await act(async () => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toBe("success");
        expect(result.current.form.value().key).toBeDefined();
      });

      const { key } = result.current.form.value();
      assert(key != null);
      const created = await client.arcs.retrieve(key);
      expect(created.name).toEqual(uniqueName);
      expect(created.source.mode).toEqual("text");
    });

    it("should create a graph-mode arc when the form picks graph", async () => {
      const { result } = renderHook(() => Arc.useForm({ query: {} }), { wrapper });
      await waitFor(() => expect(result.current.variant).toBe("success"));
      act(() => {
        result.current.form.set(path.name, `form-graph-${id.create()}`);
        result.current.form.set(path.mode, "graph");
      });
      await act(async () => {
        result.current.save();
      });
      await waitFor(() => expect(result.current.form.value().key).toBeDefined());
      const { key } = result.current.form.value();
      assert(key != null);
      const created = await client.arcs.retrieve(key);
      expect(created.source.mode).toEqual("graph");
    });

    it("should not create an arc when the name is empty", async () => {
      const { result } = renderHook(() => Arc.useForm({ query: {} }), { wrapper });

      await waitFor(() => expect(result.current.variant).toBe("success"));

      await act(async () => {
        result.current.save();
      });

      expect(result.current.form.value().key).toBeUndefined();
      expect(result.current.form.get(path.name).status.variant).toBe("error");
    });

    it("should not create a module whose name is not an identifier", async () => {
      const { result } = renderHook(() => Arc.useForm({ query: {} }), { wrapper });
      await waitFor(() => expect(result.current.variant).toBe("success"));
      act(() => {
        result.current.form.set(path.name, "bad name");
        result.current.form.set(path.type, "module");
      });
      await act(async () => {
        result.current.save();
      });
      expect(result.current.form.value().key).toBeUndefined();
      const { status } = result.current.form.get(path.name);
      expect(status.variant).toBe("error");
      expect(status.message).toContain("must start with a letter or underscore");
    });
  });

  describe("useNodeForm", () => {
    const remote = createTestClient();
    const STATUS_INPUTS = {
      type: "status.set",
      key_or_name: "alarm",
      variant: "success",
      message: "ok",
    };

    beforeEach(async () => await waitForStreamLive(client.connection));

    afterEach(() => {
      vi.restoreAllMocks();
    });

    const createStatusArc = async (): Promise<arc.Arc> =>
      await createTestArc({
        source: {
          mode: "graph",
          graph: {
            nodes: [
              { key: "n1", position: { x: 0, y: 0 } },
              { key: "n2", position: { x: 10, y: 10 } },
            ],
            edges: [],
            inputs: {
              n1: STATUS_INPUTS,
              n2: { ...STATUS_INPUTS, key_or_name: "fault" },
            },
          },
        },
      });

    const inputs = deep.path<typeof STATUS_INPUTS>();

    const renderForm = async (key: arc.Key) =>
      await renderHookSuspended(
        ({ nodeKey }: { nodeKey: string }) => Arc.useNodeForm({ key, nodeKey }),
        { wrapper, initialProps: { nodeKey: "n1" } },
      );

    // A fresh client reads the Core. A streaming client's cache can lag a dispatch.
    const fetchStoredInputs = async (key: arc.Key, nodeKey: string): Promise<unknown> =>
      graphOf(await createTestClient().arcs.retrieve({ key })).inputs[nodeKey];

    it("should show the node's inputs", async () => {
      const a = await createStatusArc();
      const { result } = await renderForm(a.key);
      expect(result.current.form.value()).toEqual(STATUS_INPUTS);
    });

    it("should send an edit as a path action that keeps input key case", async () => {
      const a = await createStatusArc();
      const { result } = await renderForm(a.key);
      const send = spyOnSend(client);
      act(() => result.current.form.set(inputs.key_or_name, "pressure"));
      await act(async () => expect(await result.current.flush()).toBe(true));
      const [, req] =
        send.mock.calls.find(([target]) => target === "/arc/dispatch") ?? [];
      expect(req).toMatchObject({
        actions: [
          {
            type: "set",
            set: { path: ["graph", "inputs", "n1", "key_or_name"], value: "pressure" },
          },
        ],
      });
      expect(await fetchStoredInputs(a.key, "n1")).toEqual({
        ...STATUS_INPUTS,
        key_or_name: "pressure",
      });
    });

    it("should show a remote edit to the node", async () => {
      const a = await createStatusArc();
      const { result } = await renderForm(a.key);
      await remote.arcs.retrieve({ key: a.key });
      await act(async () => {
        await remote.arcs.dispatch(
          a.key,
          arc.set({ path: ["graph", "inputs", "n1", "message"], value: "remote" }),
        );
      });
      await waitFor(() =>
        expect(result.current.form.get(inputs.message).value).toBe("remote"),
      );
    });

    it("should edit the new node when the node key changes", async () => {
      const a = await createStatusArc();
      const { result, rerender } = await renderForm(a.key);
      rerender({ nodeKey: "n2" });
      expect(result.current.form.get(inputs.key_or_name).value).toBe("fault");
      act(() => result.current.form.set(inputs.message, "tripped"));
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect(await fetchStoredInputs(a.key, "n2")).toMatchObject({
        message: "tripped",
      });
      expect(await fetchStoredInputs(a.key, "n1")).toEqual(STATUS_INPUTS);
    });
  });

  describe("use", () => {
    it("should retrieve a single arc", async () => {
      const testArc = await client.arcs.create({
        name: `retrieve-arc-${Math.random().toString(36).substring(7)}`,
        type: "automation",
        source: { mode: "text" },
      });

      const { result } = await renderHookSuspended(
        () => Arc.use({ key: testArc.key }),
        {
          wrapper,
        },
      );

      await waitFor(() => {
        expect(result.current).not.toBeNull();
      });

      expect(result.current?.key).toBe(testArc.key);
      expect(result.current?.name).toBe(testArc.name);
    });

    it("does not suspend when the arc is already in the store", async () => {
      const a = await createTestArc();
      await loadArc(a.key);

      const Wrapper = wrapper;
      // With the store warm, the fast-path resolves without suspending.
      const Probe = (): ReactElement => {
        Arc.useEnsure({ key: a.key });
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
        type: "automation",
        source: { mode: "text" },
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

  describe("useTaskControls", () => {
    it("should report a not-deployed status for an arc with no task", async () => {
      const testArc = await client.arcs.create({
        name: `arc-untasked-${id.create()}`,
        type: "automation",
        source: { mode: "graph", graph: { nodes: [], edges: [] } },
      });
      const { result } = renderHook(
        () => Arc.useTaskControls(testArc.key, testArc.name),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.taskStatus.message).toBe("Task not deployed"),
      );
      expect(result.current.running).toBe(false);
      expect(result.current.taskKey).toBe("");
    });
  });

  describe("useDrifted", () => {
    const createDeployed = async (): Promise<{ a: arc.Arc; tsk: task.Task }> => {
      const rck = await client.racks.create({ name: `rack-${id.create()}` });
      const a = await client.arcs.create({
        name: `arc-drift-${id.create()}`,
        type: "automation",
        source: { mode: "text" },
      });
      const tsk = await client.arcs.updateTask(a.key, { rack: rck.key });
      if (tsk == null) throw new Error("expected a deployment task");
      return { a, tsk };
    };

    const setRunning = async (
      tsk: task.Task,
      overrides: { configHash?: string; rack?: number } = {},
    ): Promise<void> => {
      await client.statuses.set(
        status.create<ReturnType<typeof task.statusDetailsZ>>({
          owner: task.resourceID(tsk.key),
          variant: "success",
          message: "Running",
          details: {
            running: true,
            cmd: "",
            configHash: overrides.configHash ?? tsk.configHash,
            rack: overrides.rack ?? tsk.rack,
            data: {},
          },
        }),
      );
    };

    const editProgram = async (key: arc.Key): Promise<void> => {
      const gen = new crdt.Text(2);
      const ops = gen.insert(0, "a -> b").map((op) => arc.insertChar(op));
      await client.arcs.dispatch(key, ops);
    };

    const renderDrifted = (a: arc.Arc) =>
      renderHook(
        () => ({
          drifted: Arc.useDrifted({ arcKey: a.key }),
          controls: Arc.useTaskControls(a.key, a.name),
        }),
        { wrapper },
      );

    it("should not drift while the task is not running", async () => {
      const { a } = await createDeployed();
      await editProgram(a.key);
      const { result } = renderDrifted(a);
      await waitFor(() => expect(result.current.controls.taskKey).not.toBe(""));
      expect(result.current.drifted).toBe(false);
    });

    it("should not drift while the deployed instance matches", async () => {
      const { a, tsk } = await createDeployed();
      await setRunning(tsk);
      const { result } = renderDrifted(a);
      await waitFor(() => expect(result.current.controls.running).toBe(true));
      expect(result.current.drifted).toBe(false);
    });

    it("should drift when the program changes after deploy", async () => {
      const { a, tsk } = await createDeployed();
      await setRunning(tsk);
      const { result } = renderDrifted(a);
      await waitFor(() => expect(result.current.controls.running).toBe(true));
      await editProgram(a.key);
      await waitFor(() => expect(result.current.drifted).toBe(true));
    });

    it("should drift when the task moves to a different rack while running", async () => {
      const { a, tsk } = await createDeployed();
      await setRunning(tsk);
      const other = await client.racks.create({ name: `rack-${id.create()}` });
      await client.arcs.updateTask(a.key, { rack: other.key });
      const { result } = renderDrifted(a);
      await waitFor(() => expect(result.current.controls.running).toBe(true));
      await waitFor(() => expect(result.current.drifted).toBe(true));
    });

    it("should not re-render when the task changes without changing drift", async () => {
      const { a, tsk } = await createDeployed();
      let renders = 0;
      const { result } = renderHook(
        () => {
          renders++;
          return Arc.useDrifted({ arcKey: a.key });
        },
        { wrapper },
      );
      // Settle the cold fetch, then edit: the task config rewrite reaches the
      // cache through the hook's subscription, but the arc is not running, so
      // the drift boolean holds false and the change must not re-render.
      await waitFor(() =>
        expect(query.isLive(client.arcs.task.watch(a.key).getCached())).toBe(true),
      );
      let settled = renders;
      await expect
        .poll(() => {
          if (renders === settled) return true;
          settled = renders;
          return false;
        })
        .toBe(true);
      await editProgram(a.key);
      await expect
        .poll(() => {
          const cached = client.arcs.task.watch(a.key).getCached();
          return query.isLive(cached) && cached?.configHash !== tsk.configHash;
        })
        .toBe(true);
      expect(result.current).toBe(false);
      expect(renders).toBe(settled);
    });
  });

  describe("selectors", () => {
    let arcKey: arc.Key;
    beforeAll(async () => {
      arcKey = (await createAndLoadArc()).key;
    });

    it("useAllNodes returns the graph nodes", () => {
      const { result } = renderHook(() => Arc.useAllNodes({ key: arcKey }), {
        wrapper,
      });
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });

    it("useNodes returns only the nodes matching the given keys", () => {
      const { result } = renderHook(() => Arc.useNodes({ key: arcKey, keys: ["n1"] }), {
        wrapper,
      });
      expect(result.current.map((n) => n.key)).toEqual(["n1"]);
    });

    it("useNodes returns an empty array when no keys are given", () => {
      const { result } = renderHook(() => Arc.useNodes({ key: arcKey, keys: [] }), {
        wrapper,
      });
      expect(result.current).toEqual([]);
    });

    it("useAllEdges returns the keyed diagram edges", () => {
      const { result } = renderHook(() => Arc.useAllEdges({ key: arcKey }), {
        wrapper,
      });
      expect(result.current).toHaveLength(1);
      const edge = result.current[0];
      expect(edge.key).toBe("edge1");
      expect(edge.source).toEqual({ node: "n1", param: "out" });
      expect(edge.target).toEqual({ node: "n2", param: "in" });
    });

    it("useNodeConfig returns the config for a node", () => {
      const { result } = renderHook(
        () => Arc.useNodeConfig({ key: arcKey, nodeKey: "n1" }),
        { wrapper },
      );
      expect(result.current).toEqual({ type: "constant", value: 0 });
    });

    it("useMode returns the representation mode", () => {
      const { result } = renderHook(() => Arc.useMode({ key: arcKey }), {
        wrapper,
      });
      expect(result.current).toBe("graph");
    });

    it("useMode reflects a text-mode arc", async () => {
      const { key } = await createAndLoadArc({ source: { mode: "text" } });
      const { result } = renderHook(() => Arc.useMode({ key }), { wrapper });
      expect(result.current).toBe("text");
    });

    it("useName returns the arc's name", async () => {
      const { key, name } = await createAndLoadArc();
      const { result } = renderHook(() => Arc.useName({ key }), { wrapper });
      expect(result.current).toBe(name);
    });
  });

  describe("selector memoization & stability", () => {
    it("useAllNodes keeps its reference when an unrelated change occurs", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          nodes: Arc.useAllNodes({ key: isolated.key }),
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

    it("useNodes keeps its reference when an unselected node moves", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          nodes: Arc.useNodes({ key: isolated.key, keys: ["n1"] }),
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

    it("useAllNodes returns a new array when a node moves", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          nodes: Arc.useAllNodes({ key: isolated.key }),
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

    it("useAllEdges keeps its transformed reference when a node moves", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          edges: Arc.useAllEdges({ key: isolated.key }),
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

    it("useAllEdges returns a new array when an edge is added", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          edges: Arc.useAllEdges({ key: isolated.key }),
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

    it("useNodeConfig keeps its reference when a different node's config changes", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          config: Arc.useNodeConfig({ key: isolated.key, nodeKey: "n1" }),
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

    it("useNodeConfig returns a new value when the node's own config changes", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          config: Arc.useNodeConfig({ key: isolated.key, nodeKey: "n1" }),
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

    it("useName keeps its value when an unrelated change occurs", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          name: Arc.useName({ key: isolated.key }),
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

    it("useName reflects a live rename", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(
        () => ({
          name: Arc.useName({ key: isolated.key }),
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
        () => Arc.useAllNodes({ key: isolated.key }),
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
          nodes: Arc.useAllNodes({ key: isolated.key }),
          config: Arc.useNodeConfig({ key: isolated.key, nodeKey: "n3" }),
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
        () => Arc.useAllNodes({ key: isolated.key }),
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
          nodes: Arc.useAllNodes({ key: isolated.key }),
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
          nodes: Arc.useAllNodes({ key: isolated.key }),
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
          nodes: Arc.useAllNodes({ key: isolated.key }),
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
          nodes: Arc.useAllNodes({ key: isolated.key }),
          config: Arc.useNodeConfig({ key: isolated.key, nodeKey: "added" }),
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
          nodes: Arc.useAllNodes({ key: isolated.key }),
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

  describe("useEnsure", () => {
    it("populates the cache so selectors resolve", async () => {
      const isolated = await createAndLoadArc();
      const { result } = renderHook(() => Arc.useAllNodes({ key: isolated.key }), {
        wrapper,
      });
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });
  });

  describe("useResultType", () => {
    it("should read the type of an Arc", async () => {
      const [automation, mod] = await Promise.all([
        createTestArc(),
        createTestArc({ type: "module" }),
      ]);
      const { result } = renderHook(
        () => ({
          automation: Arc.useResultType({ key: automation.key }),
          mod: Arc.useResultType({ key: mod.key }),
        }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.automation.data).toEqual("automation");
        expect(result.current.mod.data).toEqual("module");
      });
    });
  });

  describe("use", () => {
    it("serves the arc initially and on each change", async () => {
      const isolated = await createTestArc();
      const { result } = await renderHookSuspended(
        () => ({
          arc: Arc.use({ key: isolated.key }),
          rename: Arc.useRename(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.arc.name).toEqual(isolated.name));
      // Names are unique in a directory, and the Core outlives one run of the spec.
      const name = `renamed_${id.create()}`;
      await act(async () => {
        await result.current.rename.updateAsync({ key: isolated.key, name });
      });
      await waitFor(() => expect(result.current.arc.name).toEqual(name));
    });
  });
});
