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
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Flux } from "@/flux";
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

describe("schematic queries", () => {
  describe("useRetrieve", () => {
    it("suspends until the schematic loads, then returns it", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();

      const Display = (): ReactElement => {
        const s = Schematic.useRetrieve({ key: schem.key });
        return <div data-testid="name">{s.name}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Flux.Suspense loading={<div>loading</div>}>
              <Display />
            </Flux.Suspense>
          </Wrapper>,
        );
      });

      await waitFor(() => {
        expect(utils.queryByTestId("name")?.textContent).toBe("test_schematic");
      });
    });
  });

  describe("useEnsureRetrieved", () => {
    it("populates the store so downstream selectors resolve", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();

      const Display = (): ReactElement => {
        Schematic.useEnsureRetrieved({ key: schem.key });
        const nodes = Schematic.useSelectAllNodes({ key: schem.key });
        return <div data-testid="nodes">{nodes.map((n) => n.key).join(",")}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Flux.Suspense loading={<div>loading</div>}>
              <Display />
            </Flux.Suspense>
          </Wrapper>,
        );
      });

      await waitFor(() => {
        expect(utils.queryByTestId("nodes")?.textContent).toBe("n1,n2");
      });
    });
  });

  describe("selectors", () => {
    it("useSelectAllEdges returns the schematic's edges", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();

      const Display = (): ReactElement => {
        Schematic.useEnsureRetrieved({ key: schem.key });
        const edges = Schematic.useSelectAllEdges({ key: schem.key });
        return <div data-testid="edges">{edges.map((e) => e.key).join(",")}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Flux.Suspense loading={<div>loading</div>}>
              <Display />
            </Flux.Suspense>
          </Wrapper>,
        );
      });

      await waitFor(() => {
        expect(utils.queryByTestId("edges")?.textContent).toBe("e1");
      });
    });

    it("useSelectSnapshot returns the snapshot flag", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();

      const Display = (): ReactElement => {
        Schematic.useEnsureRetrieved({ key: schem.key });
        const snap = Schematic.useSelectSnapshot({ key: schem.key });
        return <div data-testid="snap">{String(snap)}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Flux.Suspense loading={<div>loading</div>}>
              <Display />
            </Flux.Suspense>
          </Wrapper>,
        );
      });

      await waitFor(() => {
        expect(utils.queryByTestId("snap")?.textContent).toBe("false");
      });
    });

    it("useSelectElementConfig returns a config by element key", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createTestSchematic();

      const Display = (): ReactElement => {
        Schematic.useEnsureRetrieved({ key: schem.key });
        const cfg = Schematic.useSelectElementConfig({
          key: schem.key,
          elKey: "n1",
        });
        return <div data-testid="variant">{(cfg as { variant: string }).variant}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Flux.Suspense loading={<div>loading</div>}>
              <Display />
            </Flux.Suspense>
          </Wrapper>,
        );
      });

      await waitFor(() => {
        expect(utils.queryByTestId("variant")?.textContent).toBe("tank");
      });
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

      const Display = (): ReactElement => {
        Schematic.useEnsureRetrieved({ key: schem.key });
        const nodes = Schematic.useSelectAllNodes({ key: schem.key });
        const n1 = nodes.find((n) => n.key === "n1");
        return (
          <div data-testid="pos">
            {n1?.position.x ?? "?"},{n1?.position.y ?? "?"}
          </div>
        );
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Flux.Suspense loading={<div>loading</div>}>
              <Display />
            </Flux.Suspense>
          </Wrapper>,
        );
      });

      await waitFor(() => {
        expect(utils.queryByTestId("pos")?.textContent).toBe("0,0");
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

      await waitFor(() => {
        expect(utils.queryByTestId("pos")?.textContent).toBe("100,200");
      });
    });
  });
});
