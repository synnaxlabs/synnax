// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type project, query, schematic } from "@synnaxlabs/client";
import {
  createTestClient,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { color, deep, TimeRange, uuid } from "@synnaxlabs/x";
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
import { type z } from "zod";

import { Errors } from "@/errors";
import { Flux } from "@/flux";
import { Schematic } from "@/schematic";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const path = deep.path<schematic.TankNodeConfig>();
const display = deep.path<schematic.StringDisplayNodeConfig>();
const element = deep.path<z.infer<typeof schematic.elementConfigZ>>();

const client = createTestClient();

const createTestSchematic = async (projectKey: string): Promise<schematic.Schematic> =>
  await client.schematics.create({
    project: projectKey,
    name: "test_schematic",
    nodes: [
      { key: "n1", position: { x: 0, y: 0 } },
      { key: "n2", position: { x: 10, y: 10 } },
    ],
    edges: [
      {
        key: "e1",
        source: { node: "n1", param: "out" },
        target: { node: "n2", param: "in" },
      },
    ],
    configs: { n1: { variant: "tank" }, n2: { variant: "tank" } },
  });

// Loads the schematic at `key` into the cache. Uses a single-hook bootstrap
// component so the suspending `useEnsure` is not followed by
// additional hooks; that shape trips a React 19 concurrent-replay warning.
const loadSchematic = async (
  Wrapper: FC<PropsWithChildren>,
  key: string,
): Promise<ReturnType<typeof render>> => {
  const Bootstrap = (): ReactElement => {
    Schematic.useEnsure({ key });
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
  return utils;
};

describe("schematic queries", () => {
  let Wrapper: FC<PropsWithChildren>;
  let proj: project.Project;
  beforeAll(async () => {
    [Wrapper, proj] = await Promise.all([
      createAsyncSynnaxWrapper({ client }),
      client.projects.create({ name: `project_${uuid.create()}` }),
    ]);
  });

  describe("use", () => {
    it("suspends until the schematic loads, then returns it", async () => {
      const schem = await createTestSchematic(proj.key);

      const Display = (): ReactElement => {
        const s = Schematic.use({ key: schem.key });
        return <div data-testid="name">{s.name}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary loading={null}>
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });

      await waitFor(() =>
        expect(utils.queryByTestId("name")?.textContent).toBe("test_schematic"),
      );
    });

    it("resolves synchronously without suspending when already in the store", async () => {
      const schem = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, schem.key);

      const Display = (): ReactElement => {
        const s = Schematic.use({ key: schem.key });
        return <div data-testid="name">{s.name}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary loading={<div data-testid="fallback" />}>
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });

      expect(utils.queryByTestId("fallback")).toBeNull();
      expect(utils.queryByTestId("name")?.textContent).toBe("test_schematic");
    });
  });

  describe("useMultiple", () => {
    it("should retrieve the schematics with the keys", async () => {
      const [a, b] = await Promise.all([
        createTestSchematic(proj.key),
        createTestSchematic(proj.key),
      ]);
      const { result } = await renderHookSuspended(
        () => Schematic.useMultiple({ keys: [a.key, b.key] }),
        { wrapper: Wrapper },
      );
      await waitFor(() =>
        expect(result.current?.map(({ key }) => key).sort()).toEqual(
          [a.key, b.key].sort(),
        ),
      );
    });

    it("should update when one of the schematics changes", async () => {
      const [a, b] = await Promise.all([
        createTestSchematic(proj.key),
        createTestSchematic(proj.key),
      ]);
      const { result } = await renderHookSuspended(
        () => Schematic.useMultiple({ keys: [a.key, b.key] }),
        { wrapper: Wrapper },
      );
      await waitFor(() => expect(result.current).toHaveLength(2));
      await act(async () => {
        await client.schematics.rename(a.key, "renamed_schematic");
      });
      await waitFor(() =>
        expect(result.current).toContainEqual(
          expect.objectContaining({ key: a.key, name: "renamed_schematic" }),
        ),
      );
    });
  });

  describe("useEnsure", () => {
    it("populates the store so downstream selectors resolve", async () => {
      const schem = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(() => Schematic.useAllNodes({ key: schem.key }), {
        wrapper: Wrapper,
      });
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });
  });

  describe("useTombstone", () => {
    it("reports the corpse on delete and clears it when the schematic is restored", async () => {
      const doomed = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, doomed.key);

      const { result } = renderHook(() => Schematic.useTombstone({ key: doomed.key }), {
        wrapper: Wrapper,
      });
      expect(result.current).toBeNull();

      await client.schematics.delete(doomed.key);
      await waitFor(() => expect(result.current?.name).toBe("test_schematic"));

      // Restoring re-creates from the corpse, which keeps the original key.
      // The read heals off the cache alone: no boundary reset, no refetch.
      const corpse = query.requireCorpse(
        client.schematics.watch(doomed.key).getCached(),
      );
      await client.schematics.create({ ...corpse, project: proj.key });
      await waitFor(() => expect(result.current).toBeNull());
    });
  });

  describe("selectors", () => {
    let schem: schematic.Schematic;
    beforeAll(async () => {
      schem = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, schem.key);
    });

    it("throws a DeletedError naming the corpse once the schematic is deleted", async () => {
      const doomed = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, doomed.key);
      await client.schematics.delete(doomed.key);
      await waitFor(() =>
        expect(
          query.Deleted.matches(client.schematics.watch(doomed.key).getCached()),
        ).toBe(true),
      );

      const caught: Error[] = [];
      const Fallback = ({ error }: Errors.FallbackProps): null => {
        caught.push(error);
        return null;
      };
      const BoundaryWrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <Errors.SuspenseBoundary loading={null} FallbackComponent={Fallback}>
            {children}
          </Errors.SuspenseBoundary>
        </Wrapper>
      );
      BoundaryWrapper.displayName = "BoundaryWrapper";

      renderHook(() => Schematic.useName({ key: doomed.key }), {
        wrapper: BoundaryWrapper,
        onCaughtError: () => {},
      });
      await waitFor(() => expect(caught.length).toBeGreaterThan(0));
      const error = caught[0];
      assert(Flux.DeletedError.matches(error));
      expect(error.corpseName).toBe("test_schematic");
      const corpse = query.requireCorpse(
        client.schematics.watch(doomed.key).getCached(),
      );
      expect(corpse.key).toBe(doomed.key);
    });

    it("useAllEdges returns the schematic's edges", () => {
      const { result } = renderHook(() => Schematic.useAllEdges({ key: schem.key }), {
        wrapper: Wrapper,
      });
      expect(result.current.map((e) => e.key)).toEqual(["e1"]);
    });

    it("useIsSnapshot returns the snapshot flag", () => {
      const { result } = renderHook(() => Schematic.useIsSnapshot({ key: schem.key }), {
        wrapper: Wrapper,
      });
      expect(result.current).toBe(false);
    });

    it("useElementConfig returns a config by element key", () => {
      const { result } = renderHook(
        () =>
          Schematic.useElementConfig({
            key: schem.key,
            elKey: "n1",
          }),
        { wrapper: Wrapper },
      );
      expect((result.current as { variant: string }).variant).toBe("tank");
    });

    it("useNodes returns nodes for the requested keys", () => {
      const { result } = renderHook(
        () => Schematic.useNodes({ key: schem.key, keys: ["n1", "n2"] }),
        { wrapper: Wrapper },
      );
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });

    it("useNodes omits missing keys without throwing", () => {
      const { result } = renderHook(
        () =>
          Schematic.useNodes({
            key: schem.key,
            keys: ["n1", "missing", "n2"],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });

    it("useNodes returns an empty array when keys is empty", () => {
      const { result } = renderHook(
        () => Schematic.useNodes({ key: schem.key, keys: [] }),
        { wrapper: Wrapper },
      );
      expect(result.current).toEqual([]);
    });

    it("useConfigs returns a map keyed by element key", () => {
      const { result } = renderHook(
        () => Schematic.useConfigs({ key: schem.key, keys: ["n1", "n2"] }),
        { wrapper: Wrapper },
      );
      expect(Array.from(result.current.keys())).toEqual(["n1", "n2"]);
      expect((result.current.get("n1") as { variant: string }).variant).toBe("tank");
      expect((result.current.get("n2") as { variant: string }).variant).toBe("tank");
    });

    it("useConfigs omits missing keys instead of shifting positions", () => {
      const { result } = renderHook(
        () =>
          Schematic.useConfigs({
            key: schem.key,
            keys: ["n1", "missing", "n2"],
          }),
        { wrapper: Wrapper },
      );
      expect(Array.from(result.current.keys())).toEqual(["n1", "n2"]);
      expect(result.current.has("missing")).toBe(false);
      expect((result.current.get("n2") as { variant: string }).variant).toBe("tank");
    });

    it("useConfigs returns an empty map when keys is empty", () => {
      const { result } = renderHook(
        () => Schematic.useConfigs({ key: schem.key, keys: [] }),
        { wrapper: Wrapper },
      );
      expect(result.current.size).toBe(0);
    });

    it("useParentOf maps members to their group", async () => {
      const isolated = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, isolated.key);
      const { result } = renderHook(
        () => ({
          parentOf: Schematic.useParentOf({ key: isolated.key }),
          dispatch: Schematic.useDispatch(),
        }),
        { wrapper: Wrapper },
      );
      expect(result.current.parentOf.size).toBe(0);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [
            schematic.setNode({
              node: { key: "g1", position: { x: 0, y: 0 } },
              config: { variant: "group_box", members: ["n1", "n2"] },
            }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.parentOf.get("n1")).toBe("g1");
        expect(result.current.parentOf.get("n2")).toBe("g1");
      });
    });

    it("useNodes keeps its reference when an unrelated node changes", async () => {
      const isolated = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, isolated.key);
      const { result } = renderHook(
        () => ({
          nodes: Schematic.useNodes({ key: isolated.key, keys: ["n1"] }),
          dispatch: Schematic.useDispatch(),
        }),
        { wrapper: Wrapper },
      );
      const initial = result.current.nodes;
      expect(initial.map((n) => n.key)).toEqual(["n1"]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [
            schematic.setNodePosition({ key: "n2", position: { x: 50, y: 50 } }),
          ],
        });
      });
      expect(result.current.nodes).toBe(initial);
    });

    it("useNodes returns a new array when a requested node changes", async () => {
      const isolated = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, isolated.key);
      const { result } = renderHook(
        () => ({
          nodes: Schematic.useNodes({ key: isolated.key, keys: ["n1"] }),
          dispatch: Schematic.useDispatch(),
        }),
        { wrapper: Wrapper },
      );
      const initial = result.current.nodes;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [
            schematic.setNodePosition({ key: "n1", position: { x: 99, y: 99 } }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.nodes).not.toBe(initial);
        expect(result.current.nodes[0].position).toEqual({ x: 99, y: 99 });
      });
    });

    it("useConfigs keeps its reference when an unrelated change occurs", async () => {
      const isolated = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, isolated.key);
      const { result } = renderHook(
        () => ({
          configs: Schematic.useConfigs({
            key: isolated.key,
            keys: ["n1", "n2"],
          }),
          dispatch: Schematic.useDispatch(),
        }),
        { wrapper: Wrapper },
      );
      const initial = result.current.configs;
      expect(initial.size).toBe(2);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: isolated.key,
          actions: [schematic.setNodePosition({ key: "n1", position: { x: 5, y: 5 } })],
        });
      });
      expect(result.current.configs).toBe(initial);
    });
  });

  describe("list", () => {
    it("should list the snapshots of a range", async () => {
      const rng = await client.ranges.create({
        name: "list_range",
        timeRange: new TimeRange(1n, 1000n),
      });
      const [source] = await Promise.all([
        client.schematics.create({ project: proj.key, name: "list_source" }),
        client.schematics.create({ project: proj.key, name: "list_other" }),
      ]);
      const snapshot = await client.schematics.copy({
        key: source.key,
        name: "list_snapshot",
        snapshot: true,
        range: rng.key,
      });
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.schematics, {
            name: "schematics",
            query: { ranges: [rng.key] },
          }),
        { wrapper: Wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => {
        expect(result.current.variant).toBe("success");
        expect(result.current.data).toEqual([snapshot.key]);
      });
      expect(result.current.getItem(snapshot.key)?.name).toBe("list_snapshot");
    });
  });

  describe("useSnapshot", () => {
    it("should copy the schematics into the range as snapshots", async () => {
      const source = await client.schematics.create({
        project: proj.key,
        name: "source",
      });
      const rng = await client.ranges.create({
        name: "snapshot_range",
        timeRange: new TimeRange(1n, 1000n),
      });
      const { result } = renderHook(() => Schematic.useSnapshot(), {
        wrapper: Wrapper,
      });
      await act(async () => {
        await result.current.updateAsync({
          schematics: { key: source.key, name: source.name },
          range: rng.key,
        });
      });
      const snapshots = await client.schematics.retrieve({ ranges: [rng.key] });
      expect(snapshots).toHaveLength(1);
      expect(snapshots[0].name).toBe("source (Snapshot)");
      expect(snapshots[0].snapshot).toBe(true);
      expect(snapshots[0].project).toBeUndefined();
    });
  });

  describe("useCreate", () => {
    it("creates a schematic and caches it", async () => {
      const { result } = renderHook(() => Schematic.useCreate(), {
        wrapper: Wrapper,
      });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          key,
          name: "created_schematic",
          project: proj.key,
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toBe("success");
      });
      expect(result.current.data?.name).toBe("created_schematic");
      expect(result.current.data?.project).toBe(proj.key);

      await loadSchematic(Wrapper, key);
      const { result: selected } = renderHook(
        () => ({
          name: Schematic.useName({ key }),
          snapshot: Schematic.useIsSnapshot({ key }),
        }),
        { wrapper: Wrapper },
      );
      expect(selected.current.name).toBe("created_schematic");
      expect(selected.current.snapshot).toBe(false);

      const retrieved = await client.schematics.retrieve(key);
      expect(retrieved.name).toBe("created_schematic");
    });
  });

  describe("useRename", () => {
    it("renames a schematic on the server", async () => {
      const schem = await createTestSchematic(proj.key);

      const { result } = renderHook(() => Schematic.useRename(), {
        wrapper: Wrapper,
      });

      await act(async () => {
        await result.current.updateAsync({
          key: schem.key,
          name: "renamed_schematic",
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toBe("success");
      });

      const retrieved = await client.schematics.retrieve(schem.key);
      expect(retrieved.name).toBe("renamed_schematic");
    });

    it("holds the new name in the cache once the rename resolves", async () => {
      const schem = await createTestSchematic(proj.key);
      const { result } = renderHook(() => Schematic.useRename(), { wrapper: Wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: schem.key, name: "renamed_in_cache" });
      });
      expect(client.schematics.watch(schem.key).getCached()).toMatchObject({
        name: "renamed_in_cache",
      });
    });
  });

  describe("useDelete", () => {
    it("deletes a schematic from the server", async () => {
      const schem = await createTestSchematic(proj.key);

      const { result } = renderHook(() => Schematic.useDelete(), {
        wrapper: Wrapper,
      });

      await act(async () => {
        await result.current.updateAsync(schem.key);
      });

      expect(result.current.variant).toBe("success");
      await expect(client.schematics.retrieve(schem.key)).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("useDispatch", () => {
    it("applies actions to the schematic and updates the store", async () => {
      const schem = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, schem.key);

      const { result: nodes } = renderHook(
        () => Schematic.useAllNodes({ key: schem.key }),
        { wrapper: Wrapper },
      );
      expect(nodes.current.find((n) => n.key === "n1")?.position).toEqual({
        x: 0,
        y: 0,
      });

      const { result: dispatchHook } = renderHook(() => Schematic.useDispatch(), {
        wrapper: Wrapper,
      });

      await act(async () => {
        await dispatchHook.current.dispatchAsync({
          key: schem.key,
          actions: [
            schematic.setNodePosition({
              key: "n1",
              position: { x: 100, y: 200 },
            }),
          ],
        });
      });

      await waitFor(() =>
        expect(nodes.current.find((n) => n.key === "n1")?.position).toEqual({
          x: 100,
          y: 200,
        }),
      );
    });
  });

  describe("useDispatch — edge stroke preprocess", () => {
    const EDGE: schematic.Edge = {
      key: "e2",
      source: { node: "n1", param: "out" },
      target: { node: "n2", param: "in" },
    };

    let schem: schematic.Schematic;
    let getEdgeCfg: () => schematic.ElementConfig | undefined;
    let dispatch: (...actions: schematic.Action[]) => Promise<void>;
    let cleanup: () => void;

    beforeEach(async () => {
      schem = await createTestSchematic(proj.key);
      const loadUtils = await loadSchematic(Wrapper, schem.key);
      const edge = renderHook(
        () => Schematic.useElementConfig({ key: schem.key, elKey: EDGE.key }),
        { wrapper: Wrapper },
      );
      const disp = renderHook(() => Schematic.useDispatch(), { wrapper: Wrapper });
      getEdgeCfg = () => edge.result.current;
      dispatch = async (...actions) =>
        await act(async () => {
          await disp.result.current.dispatchAsync({ key: schem.key, actions });
        });
      cleanup = () => {
        edge.unmount();
        disp.unmount();
        loadUtils.unmount();
      };
    });

    afterEach(() => cleanup());

    it("fills a new edge's stroke from its source symbol", async () => {
      await dispatch(
        schematic.setConfig({
          key: "n1",
          config: { variant: "tank", strokeColor: "#00ff00" },
        }),
      );
      await dispatch(
        schematic.addEdge({ edge: EDGE }),
        schematic.setConfig({ key: EDGE.key, config: { variant: "pipe" } }),
      );
      await waitFor(() =>
        expect(getEdgeCfg()).toMatchObject({
          variant: "pipe",
          strokeColor: [0, 255, 0, 255],
        }),
      );
    });

    it("keeps the stroke a new edge's config chose", async () => {
      await dispatch(
        schematic.setConfig({
          key: "n1",
          config: { variant: "tank", strokeColor: "#00ff00" },
        }),
      );
      await dispatch(
        schematic.addEdge({ edge: EDGE }),
        schematic.setConfig({
          key: EDGE.key,
          config: { variant: "pipe", strokeColor: "#0000ff" },
        }),
      );
      await waitFor(() =>
        expect(getEdgeCfg()).toMatchObject({
          variant: "pipe",
          strokeColor: [0, 0, 255, 255],
        }),
      );
    });

    it("leaves a new edge alone when its source symbol has no stroke", async () => {
      await dispatch(
        schematic.addEdge({ edge: EDGE }),
        schematic.setConfig({ key: EDGE.key, config: { variant: "pipe" } }),
      );
      await waitFor(() => expect(getEdgeCfg()).toMatchObject({ variant: "pipe" }));
      expect(getEdgeCfg()).not.toHaveProperty("strokeColor");
    });
  });

  describe("useDispatch — edge segment preprocess", () => {
    const SEGMENTS: Schematic.Edge.Segmented.Segment[] = [
      { direction: "x", length: 50 },
      { direction: "y", length: 30 },
      { direction: "x", length: 50 },
    ];

    interface EdgeCfg {
      variant?: string;
      strokeColor?: string;
      segments?: Schematic.Edge.Segmented.Segment[];
    }
    const asEdgeCfg = (raw: unknown): EdgeCfg | undefined => raw as EdgeCfg | undefined;

    let schem: schematic.Schematic;
    let getEdgeCfg: () => EdgeCfg | undefined;
    let getNode: (k: string) => schematic.Node | undefined;
    let dispatch: (...actions: schematic.Action[]) => Promise<void>;
    let undo: () => Promise<void>;
    let cleanup: () => void;

    beforeEach(async () => {
      schem = await createTestSchematic(proj.key);
      const loadUtils = await loadSchematic(Wrapper, schem.key);

      const edge = renderHook(
        () => Schematic.useElementConfig({ key: schem.key, elKey: "e1" }),
        { wrapper: Wrapper },
      );
      const all = renderHook(() => Schematic.useAllNodes({ key: schem.key }), {
        wrapper: Wrapper,
      });
      const disp = renderHook(() => Schematic.useDispatch(), {
        wrapper: Wrapper,
      });
      const und = renderHook(() => Schematic.useUndo({ key: schem.key }), {
        wrapper: Wrapper,
      });

      getEdgeCfg = () => asEdgeCfg(edge.result.current);
      getNode = (k) => all.result.current.find((n) => n.key === k);
      dispatch = async (...actions) =>
        await act(async () => {
          await disp.result.current.dispatchAsync({ key: schem.key, actions });
        });
      undo = async () =>
        await act(async () => {
          und.result.current.undo();
        });
      cleanup = () => {
        edge.unmount();
        all.unmount();
        disp.unmount();
        und.unmount();
        loadUtils.unmount();
      };
    });

    afterEach(() => cleanup());

    it("adjusts a connected edge's stored segments when its source node moves", async () => {
      await dispatch(
        schematic.setConfig({
          key: "e1",
          config: { variant: "pipe", segments: SEGMENTS },
        }),
      );
      await waitFor(() => expect(getEdgeCfg()?.segments).toEqual(SEGMENTS));

      await dispatch(
        schematic.setNodePosition({ key: "n1", position: { x: 20, y: 0 } }),
      );
      await waitFor(() =>
        expect(getEdgeCfg()?.segments).toEqual([
          { direction: "x", length: 30 },
          { direction: "y", length: 30 },
          { direction: "x", length: 50 },
        ]),
      );
    });

    it("writes only the segments of a connected edge", async () => {
      await dispatch(
        schematic.setConfig({
          key: "e1",
          config: { variant: "pipe", segments: SEGMENTS },
        }),
      );
      await waitFor(() => expect(getEdgeCfg()?.segments).toEqual(SEGMENTS));
      const send = spyOnSend(client);
      await dispatch(
        schematic.setNodePosition({ key: "n1", position: { x: 20, y: 0 } }),
      );
      const [, req] =
        send.mock.calls.find(([target]) => target === "/schematic/dispatch") ?? [];
      expect(req).toMatchObject({
        actions: [
          { type: "set_node_position" },
          { type: "set", set: { path: ["configs", "e1", "segments"] } },
        ],
      });
    });

    it("does not synthesize a setConfig when the edge has no stored segments", async () => {
      expect(getEdgeCfg()).toBeUndefined();

      await dispatch(
        schematic.setNodePosition({ key: "n1", position: { x: 20, y: 0 } }),
      );
      await waitFor(() => expect(getNode("n1")?.position).toEqual({ x: 20, y: 0 }));
      expect(getEdgeCfg()).toBeUndefined();
    });

    it("leaves edge segments alone when source and target move by equal deltas", async () => {
      await dispatch(
        schematic.setConfig({
          key: "e1",
          config: { variant: "pipe", segments: SEGMENTS },
        }),
      );
      await waitFor(() => expect(getEdgeCfg()?.segments).toEqual(SEGMENTS));

      // n1 at (0,0), n2 at (10,10). Shift both by (+20, +5).
      await dispatch(
        schematic.setNodePosition({ key: "n1", position: { x: 20, y: 5 } }),
        schematic.setNodePosition({ key: "n2", position: { x: 30, y: 15 } }),
      );
      await waitFor(() => {
        expect(getNode("n1")?.position).toEqual({ x: 20, y: 5 });
        expect(getNode("n2")?.position).toEqual({ x: 30, y: 15 });
      });
      expect(getEdgeCfg()?.segments).toEqual(SEGMENTS);
    });

    it("preserves non-segment edge config fields when a connected node moves", async () => {
      await dispatch(
        schematic.setConfig({
          key: "e1",
          config: { variant: "pipe", strokeColor: "#ff00ff", segments: SEGMENTS },
        }),
      );
      await waitFor(() =>
        expect(getEdgeCfg()).toMatchObject({
          variant: "pipe",
          strokeColor: [255, 0, 255, 255],
          segments: SEGMENTS,
        }),
      );

      await dispatch(
        schematic.setNodePosition({ key: "n1", position: { x: 20, y: 0 } }),
      );
      await waitFor(() => {
        const cfg = getEdgeCfg();
        expect(cfg?.variant).toBe("pipe");
        expect(cfg?.strokeColor).toEqual([255, 0, 255, 255]);
        expect(cfg?.segments?.[0]).toEqual({ direction: "x", length: 30 });
      });
    });

    it("propagates dispatched actions to a second client via sy_schematic_set", async () => {
      const clientB = createTestClient();
      const WrapperB = await createAsyncSynnaxWrapper({ client: clientB });
      await loadSchematic(WrapperB, schem.key);

      const { result: nodesB } = renderHook(
        () => Schematic.useAllNodes({ key: schem.key }),
        { wrapper: WrapperB },
      );

      await dispatch(
        schematic.setNodePosition({ key: "n1", position: { x: 50, y: 50 } }),
      );

      await waitFor(() => {
        expect(getNode("n1")?.position).toEqual({ x: 50, y: 50 });
        expect(nodesB.current.find((n) => n.key === "n1")?.position).toEqual({
          x: 50,
          y: 50,
        });
      });
    });

    it("dedups its own echo so undo fully reverts a dispatched action", async () => {
      const clientB = createTestClient();
      const WrapperB = await createAsyncSynnaxWrapper({ client: clientB });
      await loadSchematic(WrapperB, schem.key);

      const { result: nodesB } = renderHook(
        () => Schematic.useAllNodes({ key: schem.key }),
        { wrapper: WrapperB },
      );

      await dispatch(
        schematic.setNodePosition({ key: "n1", position: { x: 50, y: 50 } }),
      );
      await waitFor(() =>
        expect(nodesB.current.find((n) => n.key === "n1")?.position).toEqual({
          x: 50,
          y: 50,
        }),
      );

      await undo();
      await waitFor(() => expect(getNode("n1")?.position).toEqual({ x: 0, y: 0 }));
    });

    it("coalesces successive moves into a single undo step", async () => {
      const longSegments: Schematic.Edge.Segmented.Segment[] = [
        { direction: "x", length: 200 },
        { direction: "y", length: 30 },
        { direction: "x", length: 50 },
      ];

      await dispatch(
        schematic.setConfig({
          key: "e1",
          config: { variant: "pipe", segments: longSegments },
        }),
      );
      await waitFor(() => expect(getEdgeCfg()?.segments).toEqual(longSegments));

      await dispatch(
        schematic.setNodePosition({ key: "n1", position: { x: 20, y: 0 } }),
      );
      await dispatch(
        schematic.setNodePosition({ key: "n1", position: { x: 40, y: 0 } }),
      );

      await undo();

      await waitFor(() => {
        expect(getNode("n1")?.position).toEqual({ x: 0, y: 0 });
        expect(getEdgeCfg()?.segments).toEqual(longSegments);
      });
    });
  });

  describe("useGroup / useUngroup", () => {
    const call = async (fn: () => string[] | null): Promise<string[] | null> => {
      let out: string[] | null = null;
      await act(async () => {
        out = fn();
      });
      return out;
    };

    interface GroupCfg {
      variant?: string;
      members?: string[];
    }

    let schem: schematic.Schematic;
    let ScopedWrapper: FC<PropsWithChildren>;

    beforeEach(async () => {
      schem = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, schem.key);
      const Scoped: FC<PropsWithChildren> = ({ children }) => (
        <Wrapper>
          <Schematic.Scope.Provider value={schem.key}>
            {children}
          </Schematic.Scope.Provider>
        </Wrapper>
      );
      Scoped.displayName = "ScopedWrapper";
      ScopedWrapper = Scoped;
    });

    it("should insert a group and return the keys to select, group first", async () => {
      const { result } = renderHook(
        () => ({
          group: Schematic.useGroup(),
          nodes: Schematic.useAllNodes(),
          configs: Schematic.useAllConfigs(),
        }),
        { wrapper: ScopedWrapper },
      );
      const selection = await call(() => result.current.group(["n1", "n2"]));
      assert(selection != null);
      expect(selection.slice(1)).toEqual(["n1", "n2"]);
      const groupKey = selection[0];
      await waitFor(() => {
        const cfg = result.current.configs[groupKey] as GroupCfg | undefined;
        expect(cfg?.variant).toEqual("group_box");
        expect(cfg?.members).toEqual(["n1", "n2"]);
        const groupNode = result.current.nodes.find((n) => n.key === groupKey);
        expect(groupNode?.zIndex).toEqual(-1);
        expect(groupNode?.position).toEqual({ x: -20, y: -40 });
      });
    });

    it("should group as one undoable step", async () => {
      const { result } = renderHook(
        () => ({
          group: Schematic.useGroup(),
          configs: Schematic.useAllConfigs(),
          undo: Schematic.useUndo(),
        }),
        { wrapper: ScopedWrapper },
      );
      const selection = await call(() => result.current.group(["n1", "n2"]));
      assert(selection != null);
      const groupKey = selection[0];
      await waitFor(() => expect(result.current.configs[groupKey]).toBeDefined());
      await act(async () => {
        result.current.undo.undo();
      });
      await waitFor(() => expect(result.current.configs[groupKey]).toBeUndefined());
    });

    it("should remove the group on ungroup and return the freed members", async () => {
      const { result } = renderHook(
        () => ({
          group: Schematic.useGroup(),
          ungroup: Schematic.useUngroup(),
          nodes: Schematic.useAllNodes(),
          configs: Schematic.useAllConfigs(),
        }),
        { wrapper: ScopedWrapper },
      );
      const selection = await call(() => result.current.group(["n1", "n2"]));
      assert(selection != null);
      const groupKey = selection[0];
      await waitFor(() => expect(result.current.configs[groupKey]).toBeDefined());
      const freed = await call(() => result.current.ungroup([groupKey]));
      expect(freed).toEqual(["n1", "n2"]);
      await waitFor(() => {
        expect(result.current.configs[groupKey]).toBeUndefined();
        expect(result.current.nodes.some((n) => n.key === groupKey)).toEqual(false);
      });
    });

    it("should return null when the selection cannot be grouped", async () => {
      const { result } = renderHook(() => Schematic.useGroup(), {
        wrapper: ScopedWrapper,
      });
      const selection = await call(() => result.current(["n1"]));
      expect(selection).toBeNull();
    });

    it("should return null when ungrouping a selection with no group", async () => {
      const { result } = renderHook(() => Schematic.useUngroup(), {
        wrapper: ScopedWrapper,
      });
      const freed = await call(() => result.current(["n1"]));
      expect(freed).toBeNull();
    });
  });

  describe("useConfigForm", () => {
    const remote = createTestClient();

    beforeEach(async () => await waitForStreamLive(client.connection));

    afterEach(() => {
      vi.restoreAllMocks();
    });

    const renderForm = async (key: schematic.Key) =>
      await renderHookSuspended(
        ({ elKey }: { elKey: string }) => Schematic.useConfigForm({ key, elKey }),
        { wrapper: Wrapper, initialProps: { elKey: "n1" } },
      );

    // A fresh client reads the Core. A streaming client's cache can lag a dispatch.
    const fetchStoredConfig = async (
      key: schematic.Key,
      elKey: string,
    ): Promise<schematic.ElementConfig> =>
      (await createTestClient().schematics.retrieve({ key })).configs[elKey];

    it("should show the element's config", async () => {
      const schem = await createTestSchematic(proj.key);
      const { result } = await renderForm(schem.key);
      expect(result.current.form.value()).toMatchObject({
        variant: "tank",
        scaleVisible: false,
      });
    });

    // Most symbol forms edit at the root of the config.
    it("should add and remove fields with a write of the whole config", async () => {
      const schem = await createTestSchematic(proj.key);
      const { result } = await renderForm(schem.key);
      const strokeColor = color.construct("#ff0000");
      const add = { ...result.current.form.value(), strokeColor };
      act(() => result.current.form.set(element, add));
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect(await fetchStoredConfig(schem.key, "n1")).toHaveProperty("strokeColor");
      const remove = { ...result.current.form.value(), strokeColor: undefined };
      act(() => result.current.form.set(element, remove));
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect(await fetchStoredConfig(schem.key, "n1")).not.toHaveProperty(
        "strokeColor",
      );
    });

    it("should replace the config when a whole write changes its variant", async () => {
      const schem = await createTestSchematic(proj.key);
      const { result } = await renderForm(schem.key);
      const next = schematic.elementConfigZ.parse({ variant: "value" });
      act(() => result.current.form.set(element, next));
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect(await fetchStoredConfig(schem.key, "n1")).toMatchObject({
        variant: "value",
      });
    });

    it("should show the last of two variant changes to an edge", async () => {
      const schem = await client.schematics.create({
        project: proj.key,
        name: "edge_variants",
        nodes: [
          { key: "n1", position: { x: 0, y: 0 } },
          { key: "n2", position: { x: 10, y: 10 } },
        ],
        edges: [
          {
            key: "e1",
            source: { node: "n1", param: "out" },
            target: { node: "n2", param: "in" },
          },
        ],
        configs: {
          n1: { variant: "tank" },
          n2: { variant: "tank" },
          e1: { variant: "pipe", segments: [] },
        },
      });
      const { result } = await renderHookSuspended(
        () => Schematic.useConfigForm({ key: schem.key, elKey: "e1" }),
        { wrapper: Wrapper },
      );
      act(() => result.current.form.set(element.variant, "electric"));
      act(() => result.current.form.set(element.variant, "secondary"));
      expect(result.current.form.get(element.variant).value).toBe("secondary");
    });

    it("should send an edit as a path action into the element's config", async () => {
      const schem = await createTestSchematic(proj.key);
      const { result } = await renderForm(schem.key);
      const send = spyOnSend(client);
      act(() => result.current.form.set(path.scaleVisible, true));
      await act(async () => expect(await result.current.flush()).toBe(true));
      const [, req] =
        send.mock.calls.find(([target]) => target === "/schematic/dispatch") ?? [];
      expect(req).toMatchObject({
        actions: [
          {
            type: "set",
            set: { path: ["configs", "n1", "scale_visible"], value: true },
          },
        ],
      });
      expect(await fetchStoredConfig(schem.key, "n1")).toMatchObject({
        scaleVisible: true,
      });
    });

    it("should show a remote edit to the element", async () => {
      const schem = await createTestSchematic(proj.key);
      const { result } = await renderForm(schem.key);
      await remote.schematics.retrieve({ key: schem.key });
      await act(async () => {
        await remote.schematics.dispatch(
          schem.key,
          schematic.set({ path: ["configs", "n1", "scaleVisible"], value: true }),
        );
      });
      await waitFor(() =>
        expect(result.current.form.get(path.scaleVisible).value).toBe(true),
      );
    });

    it("should set the first entry of an empty list without keys", async () => {
      const schem = await client.schematics.create({
        project: proj.key,
        name: "test_schematic",
        nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
        configs: { n1: { variant: "string_display" } },
      });
      const { result } = await renderForm(schem.key);
      act(() => result.current.form.set(display.tooltip[0], "line"));
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect(await fetchStoredConfig(schem.key, "n1")).toMatchObject({
        tooltip: ["line"],
      });
    });

    it("should edit the new element when the element key changes", async () => {
      const schem = await createTestSchematic(proj.key);
      const { result, rerender } = await renderForm(schem.key);
      rerender({ elKey: "n2" });
      act(() => result.current.form.set(path.scaleVisible, true));
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect(await fetchStoredConfig(schem.key, "n2")).toMatchObject({
        scaleVisible: true,
      });
      expect(await fetchStoredConfig(schem.key, "n1")).toMatchObject({
        scaleVisible: false,
      });
    });
  });
});
