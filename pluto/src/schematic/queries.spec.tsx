// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createTestClient,
  NotFoundError,
  schematic,
  type workspace,
} from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Errors } from "@/errors";
import { Schematic } from "@/schematic";
import { GROUP_VARIANT } from "@/schematic/groups";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const createTestSchematic = async (wsKey: string): Promise<schematic.Schematic> =>
  await client.schematics.create(wsKey, {
    ...schematic.ZERO_NEW,
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

// Two loose nodes plus a group container (g1) holding members m1 and m2. The
// group state is set up via dispatch so groupId is round-tripped through the
// action pipeline (matching production), rather than relying on the initial
// create call to persist groupId on each node.
const createGroupedSchematic = async (
  Wrapper: FC<PropsWithChildren>,
): Promise<schematic.Schematic> => {
  const ws = await client.workspaces.create({
    name: `ws_${uuid.create()}`,
    layout: {},
  });
  const schem = await client.schematics.create(ws.key, {
    ...schematic.ZERO_NEW,
    name: "grouped_schematic",
    nodes: [
      { key: "loose1", position: { x: 0, y: 0 } },
      { key: "loose2", position: { x: 400, y: 400 } },
    ],
    edges: [],
    configs: {
      loose1: { variant: "tank" },
      loose2: { variant: "tank" },
    },
  });
  await loadSchematic(Wrapper, schem.key);
  const setup = renderHook(() => Schematic.useDispatch(), { wrapper: Wrapper });
  await act(async () => {
    await setup.result.current.dispatchAsync({
      key: schem.key,
      actions: [
        schematic.setNode({
          node: { key: "g1", position: { x: 100, y: 100 }, zIndex: -1 },
          config: {
            variant: GROUP_VARIANT,
            dimensions: { width: 200, height: 200 },
          },
        }),
        schematic.setNode({
          node: { key: "m1", position: { x: 130, y: 130 }, groupId: "g1" },
          config: { variant: "tank", label: { label: "MemberA" } },
        }),
        schematic.setNode({
          node: { key: "m2", position: { x: 200, y: 200 }, groupId: "g1" },
          config: { variant: "tank", label: { label: "MemberB" } },
        }),
      ],
    });
  });
  return schem;
};

// Populates the flux store with the schematic at `key`. Uses a single-hook
// bootstrap component so the suspending `useEnsureRetrieved` is not followed by
// additional hooks — that shape trips a React 19 concurrent-replay warning.
const loadSchematic = async (
  Wrapper: FC<PropsWithChildren>,
  key: string,
): Promise<void> => {
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
  await utils.findByTestId("loaded");
};

describe("schematic queries", () => {
  let Wrapper: FC<PropsWithChildren>;
  let ws: workspace.Workspace;
  beforeAll(async () => {
    [Wrapper, ws] = await Promise.all([
      createAsyncSynnaxWrapper({ client }),
      client.workspaces.create({ name: `ws_${uuid.create()}`, layout: {} }),
    ]);
  });

  describe("useRetrieveSuspended", () => {
    it("suspends until the schematic loads, then returns it", async () => {
      const schem = await createTestSchematic(ws.key);

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
  });

  describe("useEnsureRetrieved", () => {
    it("populates the store so downstream selectors resolve", async () => {
      const schem = await createTestSchematic(ws.key);
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
      schem = await createTestSchematic(ws.key);
      await loadSchematic(Wrapper, schem.key);
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

    it("useSelectEdge returns the edge for a known key", () => {
      const { result } = renderHook(
        () => Schematic.useSelectEdge({ key: schem.key, edgeKey: "e1" }),
        { wrapper: Wrapper },
      );
      expect(result.current?.key).toBe("e1");
    });

    it("useSelectEdge returns undefined for an unknown edge key", () => {
      const { result } = renderHook(
        () => Schematic.useSelectEdge({ key: schem.key, edgeKey: "missing" }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBeUndefined();
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
  });

  describe("useCreate", () => {
    it("creates a schematic and stores it in the flux store", async () => {
      const { result } = renderHook(() => Schematic.useCreate(), {
        wrapper: Wrapper,
      });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          ...schematic.ZERO_NEW,
          key,
          name: "created_schematic",
          workspace: ws.key,
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toBe("success");
      });
      expect(result.current.data?.name).toBe("created_schematic");
      expect(result.current.data?.workspace).toBe(ws.key);

      const retrieved = await client.schematics.retrieve({ key });
      expect(retrieved.name).toBe("created_schematic");
    });
  });

  describe("useRename", () => {
    it("renames a schematic on the server", async () => {
      const schem = await createTestSchematic(ws.key);

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

      const retrieved = await client.schematics.retrieve({ key: schem.key });
      expect(retrieved.name).toBe("renamed_schematic");
    });
  });

  describe("useDelete", () => {
    it("deletes a schematic from the server", async () => {
      const schem = await createTestSchematic(ws.key);

      const { result } = renderHook(() => Schematic.useDelete(), {
        wrapper: Wrapper,
      });

      await act(async () => {
        await result.current.updateAsync(schem.key);
      });

      expect(result.current.variant).toBe("success");
      await expect(client.schematics.retrieve({ key: schem.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("useSelectCanGroup", () => {
    it("should return true when ≥2 non-group nodes are selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic(ws.key);
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () =>
          Schematic.useSelectCanGroup({
            key: schem.key,
            selected: ["n1", "n2"],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(true);
    });

    it("should return false when only 1 non-group node is selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic(ws.key);
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () =>
          Schematic.useSelectCanGroup({
            key: schem.key,
            selected: ["n1"],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(false);
    });

    it("should return false when no nodes are selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic(ws.key);
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () =>
          Schematic.useSelectCanGroup({
            key: schem.key,
            selected: [],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(false);
    });

    it("should exclude group containers from the non-group count", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createGroupedSchematic(Wrapper);

      // g1 is a group container; loose1 is loose. Only 1 non-group → false.
      const { result } = renderHook(
        () =>
          Schematic.useSelectCanGroup({
            key: schem.key,
            selected: ["g1", "loose1"],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(false);
    });

    it("should return false for a schematic that has not been loaded", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(
        () =>
          Schematic.useSelectCanGroup({
            key: uuid.create(),
            selected: ["n1", "n2"],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(false);
    });
  });

  describe("useSelectCanUngroup", () => {
    it("should return true when a group container is selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createGroupedSchematic(Wrapper);

      const { result } = renderHook(
        () =>
          Schematic.useSelectCanUngroup({
            key: schem.key,
            selected: ["g1"],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(true);
    });

    it("should return true when a node with a groupId is selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createGroupedSchematic(Wrapper);

      const { result } = renderHook(
        () =>
          Schematic.useSelectCanUngroup({
            key: schem.key,
            selected: ["m1"],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(true);
    });

    it("should return false when selected nodes have no group association", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createGroupedSchematic(Wrapper);

      const { result } = renderHook(
        () =>
          Schematic.useSelectCanUngroup({
            key: schem.key,
            selected: ["loose1", "loose2"],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(false);
    });

    it("should return false when no nodes are selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createGroupedSchematic(Wrapper);

      const { result } = renderHook(
        () =>
          Schematic.useSelectCanUngroup({
            key: schem.key,
            selected: [],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(false);
    });

    it("should return false for a schematic that has not been loaded", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(
        () =>
          Schematic.useSelectCanUngroup({
            key: uuid.create(),
            selected: ["g1"],
          }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(false);
    });
  });

  describe("useGroup", () => {
    it("should create a new group container at the bounding box of selected members", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic(ws.key);
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => ({
          group: Schematic.useGroup(schem.key),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
          configs: Schematic.useSelectAllConfigs({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      act(() => result.current.group(["n1", "n2"]));

      await waitFor(() => expect(result.current.nodes).toHaveLength(3));
      const newGroup = result.current.nodes.find(
        (n) => n.key !== "n1" && n.key !== "n2",
      );
      expect(newGroup).toBeDefined();
      expect(
        (result.current.configs[newGroup!.key] as { variant: string }).variant,
      ).toBe(GROUP_VARIANT);
      // Members were updated with the new group's key.
      expect(result.current.nodes.find((n) => n.key === "n1")?.groupId).toBe(
        newGroup!.key,
      );
      expect(result.current.nodes.find((n) => n.key === "n2")?.groupId).toBe(
        newGroup!.key,
      );
    });

    it("should not act when fewer than 2 non-group members would be in the result", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic(ws.key);
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => ({
          group: Schematic.useGroup(schem.key),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const before = result.current.nodes.length;
      act(() => result.current.group(["n1"]));
      // No new group created; node count unchanged.
      expect(result.current.nodes).toHaveLength(before);
    });

    it("should dissolve an existing group container and merge its members into a super-group", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createGroupedSchematic(Wrapper);

      const { result } = renderHook(
        () => ({
          group: Schematic.useGroup(schem.key),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
          configs: Schematic.useSelectAllConfigs({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      // Select existing group g1 + loose1 → expect g1 dissolved, new super-group
      // contains m1, m2, loose1.
      act(() => result.current.group(["g1", "loose1"]));

      await waitFor(() => {
        const newGroup = result.current.nodes.find(
          (n) =>
            (result.current.configs[n.key] as { variant?: string } | undefined)
              ?.variant === GROUP_VARIANT,
        );
        expect(newGroup?.key).not.toBe("g1");
      });

      // g1 is gone.
      expect(result.current.nodes.find((n) => n.key === "g1")).toBeUndefined();
      // m1, m2, loose1 share the new group container's key.
      const newGroup = result.current.nodes.find(
        (n) =>
          (result.current.configs[n.key] as { variant?: string } | undefined)
            ?.variant === GROUP_VARIANT,
      );
      expect(result.current.nodes.find((n) => n.key === "m1")?.groupId).toBe(
        newGroup!.key,
      );
      expect(result.current.nodes.find((n) => n.key === "m2")?.groupId).toBe(
        newGroup!.key,
      );
      expect(result.current.nodes.find((n) => n.key === "loose1")?.groupId).toBe(
        newGroup!.key,
      );
    });
  });

  describe("useUngroup", () => {
    it("should remove a selected group container and clear its members' groupIds", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createGroupedSchematic(Wrapper);

      const { result } = renderHook(
        () => ({
          ungroup: Schematic.useUngroup(schem.key),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      act(() => result.current.ungroup(["g1"]));

      await waitFor(() =>
        expect(result.current.nodes.find((n) => n.key === "g1")).toBeUndefined(),
      );
      expect(result.current.nodes.find((n) => n.key === "m1")?.groupId).toBeUndefined();
      expect(result.current.nodes.find((n) => n.key === "m2")?.groupId).toBeUndefined();
    });

    it("should ungroup the whole group when a member is selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createGroupedSchematic(Wrapper);

      const { result } = renderHook(
        () => ({
          ungroup: Schematic.useUngroup(schem.key),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      act(() => result.current.ungroup(["m1"]));

      await waitFor(() =>
        expect(result.current.nodes.find((n) => n.key === "g1")).toBeUndefined(),
      );
      expect(result.current.nodes.find((n) => n.key === "m1")?.groupId).toBeUndefined();
      expect(result.current.nodes.find((n) => n.key === "m2")?.groupId).toBeUndefined();
    });

    it("should not act when no selected node touches a group", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createGroupedSchematic(Wrapper);

      const { result } = renderHook(
        () => ({
          ungroup: Schematic.useUngroup(schem.key),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const before = result.current.nodes.length;
      act(() => result.current.ungroup(["loose1"]));
      expect(result.current.nodes).toHaveLength(before);
      expect(result.current.nodes.find((n) => n.key === "g1")).toBeDefined();
    });
  });

  describe("group/ungroup round-trip", () => {
    it("should return the schematic to its pre-group state after group then ungroup", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic(ws.key);
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => ({
          group: Schematic.useGroup(schem.key),
          ungroup: Schematic.useUngroup(schem.key),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
          configs: Schematic.useSelectAllConfigs({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const beforeKeys = new Set(result.current.nodes.map((n) => n.key));
      const beforeCount = result.current.nodes.length;

      act(() => result.current.group(["n1", "n2"]));
      await waitFor(() => expect(result.current.nodes).toHaveLength(beforeCount + 1));

      const newGroup = result.current.nodes.find((n) => !beforeKeys.has(n.key))!;
      act(() => result.current.ungroup([newGroup.key]));

      await waitFor(() => expect(result.current.nodes).toHaveLength(beforeCount));
      // The new group container is gone.
      expect(result.current.nodes.find((n) => n.key === newGroup.key)).toBeUndefined();
      // The original members no longer carry a groupId.
      expect(result.current.nodes.find((n) => n.key === "n1")?.groupId).toBeUndefined();
      expect(result.current.nodes.find((n) => n.key === "n2")?.groupId).toBeUndefined();
      // The schematic now contains exactly the original keys.
      expect(new Set(result.current.nodes.map((n) => n.key))).toEqual(beforeKeys);
    });
  });

  describe("useDispatch", () => {
    it("applies actions to the schematic and updates the store", async () => {
      const schem = await createTestSchematic(ws.key);
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
});
