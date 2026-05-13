// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError, schematic } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Errors } from "@/errors";
import { Schematic } from "@/schematic";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const createTestSchematic = async (): Promise<schematic.Schematic> => {
  const ws = await client.workspaces.create({
    name: `ws_${uuid.create()}`,
    layout: {},
  });
  return await client.schematics.create(ws.key, {
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
  describe("useRetrieveSuspended", () => {
    it("suspends until the schematic loads, then returns it", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();

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
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useSelectAllNodes({ key: schem.key }),
        { wrapper: Wrapper },
      );
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });
  });

  describe("selectors", () => {
    it("useSelectAllEdges returns the schematic's edges", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useSelectAllEdges({ key: schem.key }),
        { wrapper: Wrapper },
      );
      expect(result.current.map((e) => e.key)).toEqual(["e1"]);
    });

    it("useSelectSnapshot returns the snapshot flag", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useSelectSnapshot({ key: schem.key }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBe(false);
    });

    it("useSelectElementConfig returns a config by element key", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

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

    it("useSelectEdge returns the edge for a known key", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useSelectEdge({ key: schem.key, edgeKey: "e1" }),
        { wrapper: Wrapper },
      );
      expect(result.current?.key).toBe("e1");
    });

    it("useSelectEdge returns undefined for an unknown edge key", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useSelectEdge({ key: schem.key, edgeKey: "missing" }),
        { wrapper: Wrapper },
      );
      expect(result.current).toBeUndefined();
    });

    it("useSelectNodes returns nodes for the requested keys", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useSelectNodes({ key: schem.key, keys: ["n1", "n2"] }),
        { wrapper: Wrapper },
      );
      expect(result.current.map((n) => n.key)).toEqual(["n1", "n2"]);
    });

    it("useSelectNodes omits missing keys without throwing", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

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

    it("useSelectNodes returns an empty array when keys is empty", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useSelectNodes({ key: schem.key, keys: [] }),
        { wrapper: Wrapper },
      );
      expect(result.current).toEqual([]);
    });

    it("useSelectConfigs returns a map keyed by element key", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useSelectConfigs({ key: schem.key, keys: ["n1", "n2"] }),
        { wrapper: Wrapper },
      );
      expect(Array.from(result.current.keys())).toEqual(["n1", "n2"]);
      expect((result.current.get("n1") as { variant: string }).variant).toBe("tank");
      expect((result.current.get("n2") as { variant: string }).variant).toBe("tank");
    });

    it("useSelectConfigs omits missing keys instead of shifting positions", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

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

    it("useSelectConfigs returns an empty map when keys is empty", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useSelectConfigs({ key: schem.key, keys: [] }),
        { wrapper: Wrapper },
      );
      expect(result.current.size).toBe(0);
    });
  });

  describe("useCreate", () => {
    it("creates a schematic and stores it in the flux store", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const ws = await client.workspaces.create({
        name: `ws_${uuid.create()}`,
        layout: {},
      });

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
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();

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
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();

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

  describe("useDispatch", () => {
    it("applies actions to the schematic and updates the store", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();
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
        await dispatchHook.current.updateAsync({
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
