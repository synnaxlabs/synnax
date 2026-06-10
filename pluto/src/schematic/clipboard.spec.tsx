// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type schematic } from "@synnaxlabs/client";
import { uuid, xy } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import {
  type ClipboardEvent as ReactClipboardEvent,
  type FC,
  type PropsWithChildren,
  type ReactElement,
} from "react";
import { describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Schematic } from "@/schematic";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

// preventDefault is a vi.fn() mock here, so passing it to expect() loses no
// `this` binding — the unbound-method warning does not apply.
/* eslint-disable @typescript-eslint/unbound-method */

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
  const ws = await client.projects.create({
    name: `ws_${uuid.create()}`,
    layout: {},
  });
  return await client.schematics.create(ws.key, {
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

describe("schematic clipboard", () => {
  describe("useClipboard", () => {
    it("writes selected nodes, edges, and configs to clipboardData on copy", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await loadSchematic(Wrapper, schem.key);

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
      await loadSchematic(Wrapper, schem.key);

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
      await loadSchematic(Wrapper, schem.key);

      const onPaste = vi.fn();
      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({
            key: schem.key,
            selected: ["n1", "n2", "e1"],
            onPaste,
          }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
          edges: Schematic.useSelectAllEdges({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const copyData = createDataTransfer();
      act(() =>
        result.current.clipboard.onCopy(createClipboardEvent(copyData), xy.ZERO),
      );
      const raw = copyData.getData(MIME);
      expect(raw).not.toBe("");

      // Paste at (200, 200): centroid was (50, 50), so offset is (150, 150).
      const pasteEvent = createClipboardEvent(createDataTransfer({ [MIME]: raw }));
      await act(async () => {
        result.current.clipboard.onPaste(pasteEvent, { x: 200, y: 200 });
      });

      await waitFor(() => expect(result.current.nodes).toHaveLength(5));

      const newNodes = result.current.nodes.filter(
        (n) => !["n1", "n2", "n3"].includes(n.key),
      );
      expect(newNodes).toHaveLength(2);
      expect(new Set(result.current.nodes.map((n) => n.key)).size).toBe(5);

      const newPositions = newNodes
        .map((n) => `${n.position.x},${n.position.y}`)
        .sort();
      expect(newPositions).toEqual(["150,150", "250,250"]);

      expect(result.current.edges.filter((e) => e.key !== "e1")).toHaveLength(1);

      expect(onPaste).toHaveBeenCalledTimes(1);
      expect(onPaste.mock.calls[0][0] as string[]).toHaveLength(2);
      expect(pasteEvent.preventDefault).toHaveBeenCalled();
    });

    it("does nothing when the clipboard has no Synnax payload", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key, selected: [] }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const event = createClipboardEvent(createDataTransfer());
      await act(async () => {
        result.current.clipboard.onPaste(event, xy.ZERO);
      });
      expect(result.current.nodes.map((n) => n.key)).toEqual(["n1", "n2", "n3"]);
      expect(event.preventDefault).not.toHaveBeenCalled();
    });

    it("ignores a payload with a mismatched version", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key, selected: [] }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
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
        result.current.clipboard.onPaste(event, xy.ZERO);
      });
      expect(result.current.nodes.map((n) => n.key)).toEqual(["n1", "n2", "n3"]);
      expect(event.preventDefault).not.toHaveBeenCalled();
    });
  });
});
