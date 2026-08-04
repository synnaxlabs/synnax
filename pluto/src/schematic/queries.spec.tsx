// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type project, query, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor, within } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { afterEach, assert, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Errors } from "@/errors";
import { Flux } from "@/flux";
import { Schematic } from "@/schematic";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const createTestSchematic = async (projectKey: string): Promise<schematic.Schematic> =>
  await client.schematics.create(projectKey, {
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
// component so the suspending `useEnsureRetrieved` is not followed by
// additional hooks; that shape trips a React 19 concurrent-replay warning.
const loadSchematic = async (
  Wrapper: FC<PropsWithChildren>,
  key: string,
): Promise<ReturnType<typeof render>> => {
  const Bootstrap = (): ReactElement => {
    Schematic.useEnsureRetrieved({ key });
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
      client.projects.create({ name: `project_${uuid.create()}`, layout: {} }),
    ]);
  });

  describe("useRetrieveSuspended", () => {
    it("suspends until the schematic loads, then returns it", async () => {
      const schem = await createTestSchematic(proj.key);

      const Display = (): ReactElement => {
        const s = Schematic.useRetrieveSuspended({ key: schem.key });
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
        const s = Schematic.useRetrieveSuspended({ key: schem.key });
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

  describe("useEnsureRetrieved", () => {
    it("populates the store so downstream selectors resolve", async () => {
      const schem = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useSelectAllNodes({ key: schem.key }),
        { wrapper: Wrapper },
      );
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
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
        expect(query.Deleted.matches(client.schematics.getCached(doomed.key))).toBe(
          true,
        ),
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

      renderHook(() => Schematic.useSelectName({ key: doomed.key }), {
        wrapper: BoundaryWrapper,
      });
      await waitFor(() => expect(caught.length).toBeGreaterThan(0));
      const error = caught[0];
      assert(Flux.DeletedError.matches(error));
      expect(error.corpseName).toBe("test_schematic");
      const corpse = query.requireCorpse(client.schematics.getCached(doomed.key));
      expect(corpse.key).toBe(doomed.key);
    });

    it("useSelectAllEdges returns the schematic's edges", () => {
      const { result } = renderHook(
        () => Schematic.useSelectAllEdges({ key: schem.key }),
        { wrapper: Wrapper },
      );
      expect(result.current.map((e) => e.key)).toEqual(["e1"]);
    });

    it("useSelectSnapshot returns the snapshot flag", () => {
      const { result } = renderHook(
        () => Schematic.useSelectSnapshot({ key: schem.key }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(false);
    });

    it("useSelectElementConfig returns a config by element key", () => {
      const { result } = renderHook(
        () =>
          Schematic.useSelectElementConfig({
            key: schem.key,
            elKey: "n1",
          }),
        { wrapper: Wrapper },
      );
      expect((result.current as { variant: string }).variant).toBe("tank");
    });

    it("useSelectNodes returns nodes for the requested keys", () => {
      const { result } = renderHook(
        () => Schematic.useSelectNodes({ key: schem.key, keys: ["n1", "n2"] }),
        { wrapper: Wrapper },
      );
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });

    it("useSelectNodes omits missing keys without throwing", () => {
      const { result } = renderHook(
        () =>
          Schematic.useSelectNodes({
            key: schem.key,
            keys: ["n1", "missing", "n2"],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });

    it("useSelectNodes returns an empty array when keys is empty", () => {
      const { result } = renderHook(
        () => Schematic.useSelectNodes({ key: schem.key, keys: [] }),
        { wrapper: Wrapper },
      );
      expect(result.current).toEqual([]);
    });

    it("useSelectConfigs returns a map keyed by element key", () => {
      const { result } = renderHook(
        () => Schematic.useSelectConfigs({ key: schem.key, keys: ["n1", "n2"] }),
        { wrapper: Wrapper },
      );
      expect(Array.from(result.current.keys())).toEqual(["n1", "n2"]);
      expect((result.current.get("n1") as { variant: string }).variant).toBe("tank");
      expect((result.current.get("n2") as { variant: string }).variant).toBe("tank");
    });

    it("useSelectConfigs omits missing keys instead of shifting positions", () => {
      const { result } = renderHook(
        () =>
          Schematic.useSelectConfigs({
            key: schem.key,
            keys: ["n1", "missing", "n2"],
          }),
        { wrapper: Wrapper },
      );
      expect(Array.from(result.current.keys())).toEqual(["n1", "n2"]);
      expect(result.current.has("missing")).toBe(false);
      expect((result.current.get("n2") as { variant: string }).variant).toBe("tank");
    });

    it("useSelectConfigs returns an empty map when keys is empty", () => {
      const { result } = renderHook(
        () => Schematic.useSelectConfigs({ key: schem.key, keys: [] }),
        { wrapper: Wrapper },
      );
      expect(result.current.size).toBe(0);
    });

    it("useSelectNodes keeps its reference when an unrelated node changes", async () => {
      const isolated = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, isolated.key);
      const { result } = renderHook(
        () => ({
          nodes: Schematic.useSelectNodes({ key: isolated.key, keys: ["n1"] }),
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

    it("useSelectNodes returns a new array when a requested node changes", async () => {
      const isolated = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, isolated.key);
      const { result } = renderHook(
        () => ({
          nodes: Schematic.useSelectNodes({ key: isolated.key, keys: ["n1"] }),
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

    it("useSelectConfigs keeps its reference when an unrelated change occurs", async () => {
      const isolated = await createTestSchematic(proj.key);
      await loadSchematic(Wrapper, isolated.key);
      const { result } = renderHook(
        () => ({
          configs: Schematic.useSelectConfigs({
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
          name: Schematic.useSelectName({ key }),
          snapshot: Schematic.useSelectSnapshot({ key }),
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
        () => Schematic.useSelectAllNodes({ key: schem.key }),
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

  describe("useDispatch — edge segment preprocess", () => {
    const SEGMENTS: Schematic.Edge.Segmented.Segment[] = [
      { direction: "x", length: 50 },
      { direction: "y", length: 30 },
      { direction: "x", length: 50 },
    ];

    interface EdgeCfg {
      variant?: string;
      color?: string;
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
        () => Schematic.useSelectElementConfig({ key: schem.key, elKey: "e1" }),
        { wrapper: Wrapper },
      );
      const all = renderHook(() => Schematic.useSelectAllNodes({ key: schem.key }), {
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
        schematic.setConfig({ key: "e1", config: { segments: SEGMENTS } }),
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
        schematic.setConfig({ key: "e1", config: { segments: SEGMENTS } }),
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
          config: { variant: "pipe", color: "#ff00ff", segments: SEGMENTS },
        }),
      );
      await waitFor(() =>
        expect(getEdgeCfg()).toMatchObject({
          variant: "pipe",
          color: "#ff00ff",
          segments: SEGMENTS,
        }),
      );

      await dispatch(
        schematic.setNodePosition({ key: "n1", position: { x: 20, y: 0 } }),
      );
      await waitFor(() => {
        const cfg = getEdgeCfg();
        expect(cfg?.variant).toBe("pipe");
        expect(cfg?.color).toBe("#ff00ff");
        expect(cfg?.segments?.[0]).toEqual({ direction: "x", length: 30 });
      });
    });

    it("propagates dispatched actions to a second client via sy_schematic_set", async () => {
      const clientB = createTestClient();
      const WrapperB = await createAsyncSynnaxWrapper({ client: clientB });
      await loadSchematic(WrapperB, schem.key);

      const { result: nodesB } = renderHook(
        () => Schematic.useSelectAllNodes({ key: schem.key }),
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
        () => Schematic.useSelectAllNodes({ key: schem.key }),
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
        schematic.setConfig({ key: "e1", config: { segments: longSegments } }),
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
});
