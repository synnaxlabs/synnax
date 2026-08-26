// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid, xy } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import {
  type ClipboardEvent as ReactClipboardEvent,
  type FC,
  type PropsWithChildren,
  type ReactElement,
} from "react";
import { assert, describe, expect, it, vi } from "vitest";

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
  const proj = await client.projects.create({
    name: `project_${uuid.create()}`,
    layout: {},
  });
  return await client.schematics.create(proj.key, {
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
// bootstrap component so the suspending `useEnsure` is not followed by
// additional hooks — that shape trips a React 19 concurrent-replay warning.
const loadSchematic = async (
  Wrapper: FC<PropsWithChildren>,
  key: string,
): Promise<void> => {
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
  await utils.findByTestId("loaded");
};

const scoped = (Wrapper: FC<PropsWithChildren>, key: string): FC<PropsWithChildren> => {
  const Scoped: FC<PropsWithChildren> = ({ children }) => (
    <Wrapper>
      <Schematic.Scope.Provider value={key}>{children}</Schematic.Scope.Provider>
    </Wrapper>
  );
  Scoped.displayName = "ScopedWrapper";
  return Scoped;
};

describe("schematic clipboard", () => {
  describe("useClipboard", () => {
    it("writes selected nodes, edges, and configs to clipboardData on copy", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useClipboard({ selected: ["n1", "n2", "e1"] }),
        { wrapper: scoped(Wrapper, schem.key) },
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

      const { result } = renderHook(() => Schematic.useClipboard({ selected: [] }), {
        wrapper: scoped(Wrapper, schem.key),
      });

      const data = createDataTransfer();
      const event = createClipboardEvent(data);
      act(() => result.current.onCopy(event, xy.ZERO));

      expect(data.getData(MIME)).toBe("");
      expect(event.preventDefault).not.toHaveBeenCalled();
    });

    it("removes the cut nodes and edges from the schematic on cut", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ selected: ["n1", "n2", "e1"] }),
          nodes: Schematic.useAllNodes({ key: schem.key }),
          edges: Schematic.useAllEdges({ key: schem.key }),
        }),
        { wrapper: scoped(Wrapper, schem.key) },
      );

      const data = createDataTransfer();
      const event = createClipboardEvent(data);
      await act(async () => {
        result.current.clipboard.onCut(event, xy.ZERO);
      });

      expect(data.getData(MIME)).not.toBe("");
      await waitFor(() =>
        expect(result.current.nodes.map((n) => n.key)).toEqual(["n3"]),
      );
      expect(result.current.edges).toHaveLength(0);
      expect(event.preventDefault).toHaveBeenCalled();
    });

    it("does nothing on cut when nothing is selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ selected: [] }),
          nodes: Schematic.useAllNodes({ key: schem.key }),
        }),
        { wrapper: scoped(Wrapper, schem.key) },
      );

      const data = createDataTransfer();
      const event = createClipboardEvent(data);
      act(() => result.current.clipboard.onCut(event, xy.ZERO));

      expect(data.getData(MIME)).toBe("");
      expect(result.current.nodes).toHaveLength(3);
      expect(event.preventDefault).not.toHaveBeenCalled();
    });

    // Chain n1 -e1-> n2 -e2-> n3 for cut tests that leave connected edges
    // unselected. Cut must remove those edges so no dangling edge persists.
    const createChainSchematic = async (): Promise<schematic.Schematic> => {
      const proj = await client.projects.create({
        name: `project_${uuid.create()}`,
        layout: {},
      });
      return await client.schematics.create(proj.key, {
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
          {
            key: "e2",
            source: { node: "n2", param: "out" },
            target: { node: "n3", param: "in" },
          },
        ],
        configs: {},
      });
    };

    const setupChain = async (selected: string[], onCut?: (keys: string[]) => void) => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createChainSchematic();
      await loadSchematic(Wrapper, schem.key);
      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ selected, onCut }),
          nodes: Schematic.useAllNodes({ key: schem.key }),
          edges: Schematic.useAllEdges({ key: schem.key }),
        }),
        { wrapper: scoped(Wrapper, schem.key) },
      );
      return result;
    };

    it("removes both unselected connected edges when cutting a middle node", async () => {
      const result = await setupChain(["n2"]);

      const data = createDataTransfer();
      const event = createClipboardEvent(data);
      await act(async () => {
        result.current.clipboard.onCut(event, xy.ZERO);
      });

      // The clipboard payload carries only the selection.
      const payload = JSON.parse(data.getData(MIME));
      expect(payload.nodes.map((n: schematic.Node) => n.key)).toEqual(["n2"]);
      expect(payload.edges).toHaveLength(0);
      await waitFor(() =>
        expect(result.current.nodes.map((n) => n.key).sort()).toEqual(["n1", "n3"]),
      );
      expect(result.current.edges).toHaveLength(0);
    });

    it("keeps edges not connected to the cut node", async () => {
      const result = await setupChain(["n1"]);

      const data = createDataTransfer();
      const event = createClipboardEvent(data);
      await act(async () => {
        result.current.clipboard.onCut(event, xy.ZERO);
      });

      await waitFor(() =>
        expect(result.current.nodes.map((n) => n.key).sort()).toEqual(["n2", "n3"]),
      );
      expect(result.current.edges.map((e) => e.key)).toEqual(["e2"]);
    });

    it("reports the surviving selection to onCut", async () => {
      const onCut = vi.fn();
      const result = await setupChain(["n2"], onCut);

      const data = createDataTransfer();
      const event = createClipboardEvent(data);
      await act(async () => {
        result.current.clipboard.onCut(event, xy.ZERO);
      });

      expect(onCut).toHaveBeenCalledWith([]);
    });

    it("cuts an edge alone without touching its endpoint nodes", async () => {
      const result = await setupChain(["e1"]);

      const data = createDataTransfer();
      const event = createClipboardEvent(data);
      await act(async () => {
        result.current.clipboard.onCut(event, xy.ZERO);
      });

      await waitFor(() =>
        expect(result.current.edges.map((e) => e.key)).toEqual(["e2"]),
      );
      expect(result.current.nodes).toHaveLength(3);
    });

    it("pastes copied nodes and edges with fresh keys at the cursor offset", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await loadSchematic(Wrapper, schem.key);

      const onPaste = vi.fn();
      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({
            selected: ["n1", "n2", "e1"],
            onPaste,
          }),
          nodes: Schematic.useAllNodes({ key: schem.key }),
          edges: Schematic.useAllEdges({ key: schem.key }),
        }),
        { wrapper: scoped(Wrapper, schem.key) },
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

    it("remaps a pasted group's members onto the pasted keys", async () => {
      interface GroupCfg {
        variant?: string;
        members?: string[];
      }
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const proj = await client.projects.create({
        name: `project_${uuid.create()}`,
        layout: {},
      });
      const schem = await client.schematics.create(proj.key, {
        name: `schem_${uuid.create()}`,
        nodes: [
          { key: "g1", position: { x: -30, y: -30 }, zIndex: -1 },
          { key: "n1", position: { x: 0, y: 0 } },
          { key: "n2", position: { x: 100, y: 100 } },
        ],
        edges: [],
        configs: {
          g1: {
            variant: "groupBox",
            members: ["n1", "n2"],
            dimensions: { width: 160, height: 160 },
          },
          n1: { variant: "tank", label: "Pump" },
          n2: { variant: "tank", label: "Valve" },
        },
      });
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ selected: ["g1", "n1", "n2"] }),
          nodes: Schematic.useAllNodes({ key: schem.key }),
          configs: Schematic.useAllConfigs({ key: schem.key }),
        }),
        { wrapper: scoped(Wrapper, schem.key) },
      );

      const copyData = createDataTransfer();
      act(() =>
        result.current.clipboard.onCopy(createClipboardEvent(copyData), xy.ZERO),
      );
      const raw = copyData.getData(MIME);
      expect(raw).not.toBe("");

      const pasteEvent = createClipboardEvent(createDataTransfer({ [MIME]: raw }));
      await act(async () => {
        result.current.clipboard.onPaste(pasteEvent, { x: 500, y: 500 });
      });
      await waitFor(() => expect(result.current.nodes).toHaveLength(6));

      const originals = new Set(["g1", "n1", "n2"]);
      const pastedKeys = result.current.nodes
        .map((n) => n.key)
        .filter((k) => !originals.has(k));
      const pastedGroupKey = pastedKeys.find(
        (k) =>
          (result.current.configs[k] as GroupCfg | undefined)?.variant === "groupBox",
      );
      assert(pastedGroupKey != null);
      const pasted = result.current.configs[pastedGroupKey] as GroupCfg;
      assert(pasted.members != null);
      expect(pasted.members).toHaveLength(2);
      expect(pasted.members.every((m) => pastedKeys.includes(m))).toEqual(true);
      expect(pasted.members.some((m) => originals.has(m))).toEqual(false);
    });

    it("does nothing when the clipboard has no Synnax payload", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ selected: [] }),
          nodes: Schematic.useAllNodes({ key: schem.key }),
        }),
        { wrapper: scoped(Wrapper, schem.key) },
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
          clipboard: Schematic.useClipboard({ selected: [] }),
          nodes: Schematic.useAllNodes({ key: schem.key }),
        }),
        { wrapper: scoped(Wrapper, schem.key) },
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
