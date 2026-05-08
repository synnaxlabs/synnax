// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, schematic } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Schematic } from "@/schematic";
import * as clipboard from "@/schematic/clipboard";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

class StringBlob {
  private readonly value: string;
  readonly type: string;
  constructor(parts: string[], options: { type: string }) {
    this.value = parts.join("");
    this.type = options.type;
  }
  async text(): Promise<string> {
    return this.value;
  }
}

class MockClipboardItem {
  readonly types: string[];
  private readonly blobs: Map<string, StringBlob>;
  constructor(items: Record<string, StringBlob>) {
    this.blobs = new Map(Object.entries(items));
    this.types = Array.from(this.blobs.keys());
  }
  async getType(type: string): Promise<StringBlob> {
    const blob = this.blobs.get(type);
    if (blob == null) throw new Error(`type not found: ${type}`);
    return blob;
  }
}

const installClipboardMock = () => {
  const buffer: MockClipboardItem[] = [];
  const write = vi.fn(async (items: MockClipboardItem[]) => {
    buffer.length = 0;
    buffer.push(...items);
  });
  const read = vi.fn(async () => buffer.slice());
  vi.stubGlobal("ClipboardItem", MockClipboardItem);
  vi.stubGlobal("Blob", StringBlob);
  Object.defineProperty(globalThis.navigator, "clipboard", {
    configurable: true,
    value: { read, write },
  });
  return {
    write,
    read,
    seed: (item: MockClipboardItem) => {
      buffer.length = 0;
      buffer.push(item);
    },
    clear: () => {
      buffer.length = 0;
    },
  };
};

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
  let mock: ReturnType<typeof installClipboardMock>;

  beforeEach(() => {
    mock = installClipboardMock();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe("encoding", () => {
    it("round-trips a payload through write and read", async () => {
      await clipboard.write(
        [{ key: "a", position: { x: 1, y: 2 } }],
        [
          {
            key: "e",
            source: { node: "a", param: "out" },
            target: { node: "b", param: "in" },
          },
        ],
        { a: { color: "red" } },
        { x: 1, y: 2 },
      );
      const got = await clipboard.read();
      expect(got).not.toBeNull();
      expect(got?.nodes).toEqual([{ key: "a", position: { x: 1, y: 2 } }]);
      expect(got?.edges[0].key).toBe("e");
      expect(got?.configs.a).toEqual({ color: "red" });
      expect(got?.anchor).toEqual({ x: 1, y: 2 });
    });

    it("returns null when the clipboard does not contain a Synnax payload", async () => {
      mock.clear();
      expect(await clipboard.read()).toBeNull();
    });

    it("returns null when the version does not match", async () => {
      mock.seed(
        new MockClipboardItem({
          "application/synnax-schematic+json": new StringBlob(
            [JSON.stringify({ version: 999, nodes: [], edges: [], configs: {} })],
            { type: "application/synnax-schematic+json" },
          ),
        }),
      );
      expect(await clipboard.read()).toBeNull();
    });
  });

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

    it("copies the selected nodes, edges, and configs", async () => {
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

      act(() => result.current.copy());
      await waitFor(() => expect(mock.write).toHaveBeenCalled());

      const payload = await clipboard.read();
      expect(payload?.nodes.map((n) => n.key).sort()).toEqual(["n1", "n2"]);
      expect(payload?.edges.map((e) => e.key)).toEqual(["e1"]);
      expect(payload?.configs.n1).toEqual({ variant: "tank", label: "Pump" });
      expect(payload?.configs.e1).toEqual({ variant: "pipe" });
      // Centroid of n1 (0,0) and n2 (100,100) = (50,50).
      expect(payload?.anchor).toEqual({ x: 50, y: 50 });
    });

    it("does nothing when nothing is selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await renderEnsuredSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useClipboard({ key: schem.key, selected: [] }),
        { wrapper: Wrapper },
      );
      act(() => result.current.copy());
      expect(mock.write).not.toHaveBeenCalled();
    });

    it("pastes copied nodes and edges with fresh keys at the target offset", async () => {
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

      act(() => result.current.copy());
      await waitFor(() => expect(mock.write).toHaveBeenCalled());

      // Paste at (200, 200): centroid was (50, 50), so offset is (150, 150).
      await act(async () => {
        await result.current.paste({ x: 200, y: 200 });
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
    });

    it("does nothing when the clipboard has no Synnax payload", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      const utils = await renderEnsuredSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => Schematic.useClipboard({ key: schem.key, selected: [] }),
        { wrapper: Wrapper },
      );
      mock.clear();
      await act(async () => {
        await result.current.paste({ x: 0, y: 0 });
      });
      expect(utils.queryByTestId("nodes")?.textContent).toBe("n1,n2,n3");
    });
  });
});
