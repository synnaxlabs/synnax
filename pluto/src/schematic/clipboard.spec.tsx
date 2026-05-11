/* eslint-disable @typescript-eslint/unbound-method */
// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, schematic } from "@synnaxlabs/client";
import { uuid, xy } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type ClipboardEvent as ReactClipboardEvent, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Schematic } from "@/schematic";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const createDataTransfer = (initial: Record<string, string> = {}): DataTransfer => {
  const store: Record<string, string> = { ...initial };
  return {
    getData: (type: string) => store[type] ?? "",
    setData: (type: string, value: string) => {
      store[type] = value;
    },
  } as unknown as DataTransfer;
};

const createClipboardEvent = (
  data: DataTransfer,
): ReactClipboardEvent<HTMLDivElement> =>
  ({
    clipboardData: data,
    preventDefault: vi.fn(),
  }) as unknown as ReactClipboardEvent<HTMLDivElement>;

const MIME = "web application/synnax-schematic+json";

const client = createTestClient();

const createSchematicWithGraph = async (): Promise<schematic.Schematic> => {
  const ws = await client.workspaces.create({
    name: `ws_${uuid.create()}`,
    layout: {},
  });
  return await client.schematics.create(ws.key, {
    ...schematic.ZERO_NEW,
    name: `schem_${uuid.create()}`,
    nodes: [
      { key: "n1", position: { x: 0, y: 0 } },
      { key: "n2", position: { x: 100, y: 100 } },
      { key: "n3", position: { x: 200, y: 200 } },
    ],
    edges: [
      {
        key: "e1",
        source: { node: "n1", param: "out" },
        target: { node: "n2", param: "in" },
      },
    ],
    configs: {
      n1: { variant: "tank", label: "Pump" },
      n2: { variant: "tank", label: "Valve" },
      n3: { variant: "tank", label: "Tank" },
      e1: { variant: "pipe" },
    },
  });
};

describe("schematic clipboard", () => {
  describe("useClipboard", () => {
    const renderEnsuredSchematic = async (
      Wrapper: Awaited<ReturnType<typeof createAsyncSynnaxWrapper>>,
      key: string,
    ) => {
      const Display = (): ReactElement => {
        Schematic.useEnsureRetrieved({ key });
        const nodes = Schematic.useSelectAllNodes({ key });
        const edges = Schematic.useSelectAllEdges({ key });
        return (
          <div>
            <div data-testid="nodes">{nodes.map((n) => n.key).join(",")}</div>
            <div data-testid="edges">{edges.map((e) => e.key).join(",")}</div>
            <div data-testid="positions">
              {nodes.map((n) => `${n.key}:${n.position.x},${n.position.y}`).join("|")}
            </div>
          </div>
        );
      };
      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary loading={<div>loading</div>}>
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });
      await waitFor(() =>
        expect(utils.queryByTestId("nodes")?.textContent).toBe("n1,n2,n3"),
      );
      return utils;
    };

    it("writes selected nodes, edges, and configs to clipboardData on copy", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await renderEnsuredSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () =>
          Schematic.useClipboard({
            key: schem.key,
            selected: ["n1", "n2", "e1"],
          }),
        { wrapper: Wrapper },
      );

      const data = createDataTransfer();
      const event = createClipboardEvent(data);
      act(() => result.current.onCopy(event, xy.ZERO));

      const raw = data.getData(MIME);
      expect(raw).not.toBe("");
      const payload = JSON.parse(raw);
      expect(payload.nodes.map((n: schematic.Node) => n.key).sort()).toEqual([
        "n1",
        "n2",
      ]);
      expect(payload.edges.map((e: schematic.Edge) => e.key)).toEqual(["e1"]);
      expect(payload.configs.n1).toEqual({ variant: "tank", label: "Pump" });
      expect(payload.configs.e1).toEqual({ variant: "pipe" });
      // Centroid of n1 (0,0) and n2 (100,100) = (50,50).
      expect(payload.anchor).toEqual({ x: 50, y: 50 });
      expect(event.preventDefault).toHaveBeenCalled();
    });

    it("does nothing on copy when nothing is selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await renderEnsuredSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useClipboard({ key: schem.key, selected: [] }),
        { wrapper: Wrapper },
      );

      const data = createDataTransfer();
      const event = createClipboardEvent(data);
      act(() => result.current.onCopy(event, xy.ZERO));

      expect(data.getData(MIME)).toBe("");
      expect(event.preventDefault).not.toHaveBeenCalled();
    });

    it("pastes copied nodes and edges with fresh keys at the cursor offset", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      const utils = await renderEnsuredSchematic(Wrapper, schem.key);

      const onPaste = vi.fn();
      const { result } = renderHook(
        () =>
          Schematic.useClipboard({
            key: schem.key,
            selected: ["n1", "n2", "e1"],
            onPaste,
          }),
        { wrapper: Wrapper },
      );

      const copyData = createDataTransfer();
      act(() => result.current.onCopy(createClipboardEvent(copyData), xy.ZERO));
      const raw = copyData.getData(MIME);
      expect(raw).not.toBe("");

      // Paste at (200, 200): centroid was (50, 50), so offset is (150, 150).
      const pasteEvent = createClipboardEvent(createDataTransfer({ [MIME]: raw }));
      await act(async () => {
        result.current.onPaste(pasteEvent, { x: 200, y: 200 });
      });

      await waitFor(() => {
        const nodeText = utils.queryByTestId("nodes")?.textContent ?? "";
        expect(nodeText.split(",")).toHaveLength(5);
      });

      const nodeKeys = (utils.queryByTestId("nodes")?.textContent ?? "").split(",");
      const newNodeKeys = nodeKeys.filter((k) => !["n1", "n2", "n3"].includes(k));
      expect(newNodeKeys).toHaveLength(2);
      expect(new Set(nodeKeys).size).toBe(nodeKeys.length);

      const positions = (utils.queryByTestId("positions")?.textContent ?? "").split(
        "|",
      );
      const newPositions = positions
        .filter((p) => newNodeKeys.some((k) => p.startsWith(`${k}:`)))
        .map((p) => p.split(":")[1])
        .sort();
      expect(newPositions).toEqual(["150,150", "250,250"]);

      const edgeKeys = (utils.queryByTestId("edges")?.textContent ?? "").split(",");
      expect(edgeKeys.filter((k) => k !== "e1")).toHaveLength(1);

      expect(onPaste).toHaveBeenCalledTimes(1);
      expect(onPaste.mock.calls[0][0] as string[]).toHaveLength(2);
      expect(pasteEvent.preventDefault).toHaveBeenCalled();
    });

    it("does nothing when the clipboard has no Synnax payload", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      const utils = await renderEnsuredSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useClipboard({ key: schem.key, selected: [] }),
        { wrapper: Wrapper },
      );

      const event = createClipboardEvent(createDataTransfer());
      await act(async () => {
        result.current.onPaste(event, xy.ZERO);
      });
      expect(utils.queryByTestId("nodes")?.textContent).toBe("n1,n2,n3");
      expect(event.preventDefault).not.toHaveBeenCalled();
    });

    it("ignores a payload with a mismatched version", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      const utils = await renderEnsuredSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useClipboard({ key: schem.key, selected: [] }),
        { wrapper: Wrapper },
      );

      const event = createClipboardEvent(
        createDataTransfer({
          [MIME]: JSON.stringify({
            version: 999,
            nodes: [],
            edges: [],
            configs: {},
            anchor: { x: 0, y: 0 },
          }),
        }),
      );
      await act(async () => {
        result.current.onPaste(event, xy.ZERO);
      });
      expect(utils.queryByTestId("nodes")?.textContent).toBe("n1,n2,n3");
      expect(event.preventDefault).not.toHaveBeenCalled();
    });
  });
});
