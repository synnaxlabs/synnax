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
import {
  type ClipboardEvent as ReactClipboardEvent,
  type FC,
  type PropsWithChildren,
  type ReactElement,
} from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Schematic } from "@/schematic";
import { GROUP_VARIANT } from "@/schematic/groups";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

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

// One group container (g1) holding members m1 and m2, plus a loose node. The
// group structure is dispatched after creation so groupId round-trips through
// the action pipeline.
const createSchematicWithGroup = async (
  Wrapper: FC<PropsWithChildren>,
): Promise<schematic.Schematic> => {
  const ws = await client.workspaces.create({
    name: `ws_${uuid.create()}`,
    layout: {},
  });
  const schem = await client.schematics.create(ws.key, {
    ...schematic.ZERO_NEW,
    name: `grouped_${uuid.create()}`,
    nodes: [{ key: "loose", position: { x: 400, y: 400 } }],
    edges: [],
    configs: { loose: { variant: "tank", label: { label: "Loose" } } },
  });
  await loadSchematic(Wrapper, schem.key);
  const setup = renderHook(() => Schematic.useDispatch(), { wrapper: Wrapper });
  await act(async () => {
    await setup.result.current.dispatchAsync({
      key: schem.key,
      actions: [
        schematic.setNode({
          node: { key: "g1", position: { x: 0, y: 0 }, zIndex: -1 },
          config: {
            variant: GROUP_VARIANT,
            dimensions: { width: 200, height: 200 },
          },
        }),
        schematic.setNode({
          node: { key: "m1", position: { x: 30, y: 30 }, groupId: "g1" },
          config: { variant: "tank", label: { label: "MemberA" } },
        }),
        schematic.setNode({
          node: { key: "m2", position: { x: 100, y: 100 }, groupId: "g1" },
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

    it("onCut removes selected nodes and edges and fills the clipboard", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGraph();
      await loadSchematic(Wrapper, schem.key);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({
            key: schem.key,
            selected: ["n1", "n2", "e1"],
          }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
          edges: Schematic.useSelectAllEdges({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const data = createDataTransfer();
      const event = createClipboardEvent(data);
      await act(async () => {
        result.current.clipboard.onCut(event, xy.ZERO);
      });

      await waitFor(() => expect(result.current.nodes).toHaveLength(1));
      expect(result.current.nodes.map((n) => n.key)).toEqual(["n3"]);
      expect(result.current.edges).toHaveLength(0);

      const raw = data.getData(MIME);
      expect(raw).not.toBe("");
      const payload = JSON.parse(raw);
      expect(payload.nodes.map((n: schematic.Node) => n.key).sort()).toEqual([
        "n1",
        "n2",
      ]);
      expect(payload.edges.map((e: schematic.Edge) => e.key)).toEqual(["e1"]);
      expect(event.preventDefault).toHaveBeenCalled();
    });
  });

  describe("group-aware clipboard", () => {
    it("onCopy should pull unselected members when the group container is selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () =>
          Schematic.useClipboard({
            key: schem.key,
            selected: ["g1"],
          }),
        { wrapper: Wrapper },
      );

      const data = createDataTransfer();
      act(() => result.current.onCopy(createClipboardEvent(data), xy.ZERO));

      const payload = JSON.parse(data.getData(MIME));
      expect(payload.nodes.map((n: schematic.Node) => n.key).sort()).toEqual([
        "g1",
        "m1",
        "m2",
      ]);
    });

    it("onCopy should pull the group container when a single member is selected", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => Schematic.useClipboard({ key: schem.key, selected: ["m1"] }),
        { wrapper: Wrapper },
      );

      const data = createDataTransfer();
      act(() => result.current.onCopy(createClipboardEvent(data), xy.ZERO));

      const payload = JSON.parse(data.getData(MIME));
      expect(payload.nodes.map((n: schematic.Node) => n.key).sort()).toEqual([
        "g1",
        "m1",
        "m2",
      ]);
    });

    it("onCopy should include configs for the group container and all members", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => Schematic.useClipboard({ key: schem.key, selected: ["g1"] }),
        { wrapper: Wrapper },
      );

      const data = createDataTransfer();
      act(() => result.current.onCopy(createClipboardEvent(data), xy.ZERO));

      const payload = JSON.parse(data.getData(MIME));
      expect(payload.configs.g1.variant).toBe(GROUP_VARIANT);
      expect(payload.configs.m1).toMatchObject({
        variant: "tank",
        label: { label: "MemberA" },
      });
      expect(payload.configs.m2).toMatchObject({
        variant: "tank",
        label: { label: "MemberB" },
      });
    });

    it("onCopy should not pull members of unselected groups", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => Schematic.useClipboard({ key: schem.key, selected: ["loose"] }),
        { wrapper: Wrapper },
      );

      const data = createDataTransfer();
      act(() => result.current.onCopy(createClipboardEvent(data), xy.ZERO));

      const payload = JSON.parse(data.getData(MIME));
      expect(payload.nodes.map((n: schematic.Node) => n.key)).toEqual(["loose"]);
    });

    it("onCut should cascade-remove the group container and all its members", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key, selected: ["g1"] }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const data = createDataTransfer();
      await act(async () => {
        result.current.clipboard.onCut(createClipboardEvent(data), xy.ZERO);
      });

      await waitFor(() => expect(result.current.nodes).toHaveLength(1));
      expect(result.current.nodes.map((n) => n.key)).toEqual(["loose"]);
      const payload = JSON.parse(data.getData(MIME));
      expect(payload.nodes.map((n: schematic.Node) => n.key).sort()).toEqual([
        "g1",
        "m1",
        "m2",
      ]);
    });

    it("onPaste should remap groupId references so pasted members point at the new container", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key, selected: ["g1"] }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
          configs: Schematic.useSelectAllConfigs({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const copyData = createDataTransfer();
      act(() =>
        result.current.clipboard.onCopy(createClipboardEvent(copyData), xy.ZERO),
      );
      const raw = copyData.getData(MIME);

      const pasteEvent = createClipboardEvent(createDataTransfer({ [MIME]: raw }));
      await act(async () => {
        result.current.clipboard.onPaste(pasteEvent, { x: 500, y: 500 });
      });

      await waitFor(() => expect(result.current.nodes).toHaveLength(7));

      // Find the new group container (the new node whose variant is GROUP_VARIANT
      // and whose key isn't g1).
      const newGroup = result.current.nodes.find(
        (n) =>
          n.key !== "g1" &&
          (result.current.configs[n.key] as { variant?: string } | undefined)
            ?.variant === GROUP_VARIANT,
      );
      expect(newGroup).toBeDefined();

      // The new members should reference the new container's key, not "g1".
      const newMembers = result.current.nodes.filter(
        (n) => n.groupId === newGroup!.key,
      );
      expect(newMembers).toHaveLength(2);
      // No new node should still reference the old "g1" key.
      const stillOldGroup = result.current.nodes
        .filter((n) => !["g1", "m1", "m2", "loose"].includes(n.key))
        .filter((n) => n.groupId === "g1");
      expect(stillOldGroup).toEqual([]);
    });

    it("onPaste should report only the newly-pasted keys to the caller (selection sync)", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const onPaste = vi.fn();
      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({
            key: schem.key,
            selected: ["g1"],
            onPaste,
          }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const copyData = createDataTransfer();
      act(() =>
        result.current.clipboard.onCopy(createClipboardEvent(copyData), xy.ZERO),
      );
      const raw = copyData.getData(MIME);

      await act(async () => {
        result.current.clipboard.onPaste(
          createClipboardEvent(createDataTransfer({ [MIME]: raw })),
          { x: 500, y: 500 },
        );
      });

      await waitFor(() => expect(onPaste).toHaveBeenCalledTimes(1));
      const reported = onPaste.mock.calls[0][0] as string[];
      const originals = new Set(["g1", "m1", "m2", "loose"]);
      expect(reported).toHaveLength(3);
      for (const k of reported) expect(originals.has(k)).toBe(false);
      const allKeys = new Set(result.current.nodes.map((n) => n.key));
      for (const k of reported) expect(allKeys.has(k)).toBe(true);
    });

    it("onPaste should generate fresh keys for every pasted node", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const onPaste = vi.fn();
      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({
            key: schem.key,
            selected: ["g1"],
            onPaste,
          }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const copyData = createDataTransfer();
      act(() =>
        result.current.clipboard.onCopy(createClipboardEvent(copyData), xy.ZERO),
      );
      const raw = copyData.getData(MIME);

      await act(async () => {
        result.current.clipboard.onPaste(
          createClipboardEvent(createDataTransfer({ [MIME]: raw })),
          { x: 500, y: 500 },
        );
      });

      await waitFor(() => expect(result.current.nodes).toHaveLength(7));
      // onPaste callback gets the new keys; none should overlap the original.
      expect(onPaste).toHaveBeenCalledTimes(1);
      const newKeys = onPaste.mock.calls[0][0] as string[];
      expect(newKeys).toHaveLength(3);
      const originals = new Set(["g1", "m1", "m2", "loose"]);
      for (const k of newKeys) expect(originals.has(k)).toBe(false);
    });

    it("cut then paste should preserve the group structure", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key, selected: ["g1"] }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
          configs: Schematic.useSelectAllConfigs({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const cutData = createDataTransfer();
      await act(async () => {
        result.current.clipboard.onCut(createClipboardEvent(cutData), xy.ZERO);
      });
      await waitFor(() => expect(result.current.nodes).toHaveLength(1));

      const raw = cutData.getData(MIME);
      const pasteEvent = createClipboardEvent(createDataTransfer({ [MIME]: raw }));
      await act(async () => {
        result.current.clipboard.onPaste(pasteEvent, { x: 500, y: 500 });
      });

      await waitFor(() => expect(result.current.nodes).toHaveLength(4));
      const newGroup = result.current.nodes.find(
        (n) =>
          (result.current.configs[n.key] as { variant?: string } | undefined)
            ?.variant === GROUP_VARIANT,
      );
      expect(newGroup).toBeDefined();
      const members = result.current.nodes.filter((n) => n.groupId === newGroup!.key);
      expect(members).toHaveLength(2);
    });

    it("onPaste twice should produce two independent groups with their own members", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key, selected: ["g1"] }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
          configs: Schematic.useSelectAllConfigs({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      const copyData = createDataTransfer();
      act(() =>
        result.current.clipboard.onCopy(createClipboardEvent(copyData), xy.ZERO),
      );
      const raw = copyData.getData(MIME);

      await act(async () => {
        result.current.clipboard.onPaste(
          createClipboardEvent(createDataTransfer({ [MIME]: raw })),
          { x: 500, y: 500 },
        );
      });
      await act(async () => {
        result.current.clipboard.onPaste(
          createClipboardEvent(createDataTransfer({ [MIME]: raw })),
          { x: 800, y: 800 },
        );
      });

      // Start: loose + g1 + m1 + m2 = 4. Each paste adds 3 (group + 2 members).
      await waitFor(() => expect(result.current.nodes).toHaveLength(10));

      const pastedGroups = result.current.nodes.filter(
        (n) =>
          n.key !== "g1" &&
          (result.current.configs[n.key] as { variant?: string } | undefined)
            ?.variant === GROUP_VARIANT,
      );
      expect(pastedGroups).toHaveLength(2);

      for (const g of pastedGroups) {
        const members = result.current.nodes.filter((n) => n.groupId === g.key);
        expect(members).toHaveLength(2);
      }
    });
  });

  describe("imperative clipboard (navigator.clipboard)", () => {
    // jsdom's Blob lacks .text(); stub Blob and ClipboardItem so the source's
    // writeAsync/readAsync can roundtrip without a real browser.
    class FakeBlob {
      readonly type: string;
      private readonly content: string;
      constructor(parts: Array<string | object>, opts?: { type?: string }) {
        this.content = parts.map((p) => (typeof p === "string" ? p : "")).join("");
        this.type = opts?.type ?? "";
      }
      async text(): Promise<string> {
        return this.content;
      }
    }

    class FakeClipboardItem {
      readonly types: string[];
      private readonly blobs: Record<string, FakeBlob>;
      constructor(blobs: Record<string, FakeBlob>) {
        this.blobs = blobs;
        this.types = Object.keys(blobs);
      }
      async getType(type: string): Promise<FakeBlob> {
        return this.blobs[type];
      }
    }

    const buildReadItem = (payload: unknown): FakeClipboardItem =>
      new FakeClipboardItem({
        [MIME]: new FakeBlob([JSON.stringify(payload)], { type: MIME }),
      });

    let writeMock: ReturnType<typeof vi.fn>;
    let readMock: ReturnType<typeof vi.fn>;
    let readResult: FakeClipboardItem[];

    beforeEach(() => {
      readResult = [];
      writeMock = vi.fn(async () => {});
      readMock = vi.fn(async () => readResult);
      Object.defineProperty(navigator, "clipboard", {
        value: { write: writeMock, read: readMock },
        configurable: true,
        writable: true,
      });
      vi.stubGlobal("ClipboardItem", FakeClipboardItem);
      vi.stubGlobal("Blob", FakeBlob);
    });

    afterEach(() => {
      vi.unstubAllGlobals();
      delete (navigator as { clipboard?: unknown }).clipboard;
    });

    it("copy writes a ClipboardItem populating both MIME and text/plain slots", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => Schematic.useClipboard({ key: schem.key, selected: ["g1"] }),
        { wrapper: Wrapper },
      );

      act(() => result.current.copy());
      await waitFor(() => expect(writeMock).toHaveBeenCalledTimes(1));

      const items = writeMock.mock.calls[0][0] as FakeClipboardItem[];
      expect(items).toHaveLength(1);
      expect(items[0].types.sort()).toEqual([MIME, "text/plain"].sort());

      const json = JSON.parse(await (await items[0].getType(MIME)).text());
      expect(json.nodes.map((n: schematic.Node) => n.key).sort()).toEqual([
        "g1",
        "m1",
        "m2",
      ]);

      const summary = await (await items[0].getType("text/plain")).text();
      expect(summary).toBe("3 nodes, 0 edges");
    });

    it("cut writes the ClipboardItem and removes the selected group and members", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key, selected: ["g1"] }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      act(() => result.current.clipboard.cut());
      await waitFor(() => expect(writeMock).toHaveBeenCalledTimes(1));
      await waitFor(() => expect(result.current.nodes).toHaveLength(1));
      expect(result.current.nodes.map((n) => n.key)).toEqual(["loose"]);
    });

    it("paste reads from the MIME slot and applies the payload at the cursor", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const onPaste = vi.fn();
      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key, onPaste }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      readResult = [
        buildReadItem({
          version: 1,
          nodes: [{ key: "src", position: { x: 0, y: 0 } }],
          edges: [],
          configs: { src: { variant: "tank" } },
          anchor: { x: 0, y: 0 },
        }),
      ];

      act(() => result.current.clipboard.paste({ x: 50, y: 50 }));
      await waitFor(() => expect(result.current.nodes).toHaveLength(5));
      expect(readMock).toHaveBeenCalledTimes(1);
      expect(onPaste).toHaveBeenCalledTimes(1);
    });

    it("paste ignores ClipboardItems that lack the schematic MIME slot", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      readResult = [
        new FakeClipboardItem({
          "text/plain": new FakeBlob(["unrelated"], { type: "text/plain" }),
        }),
      ];

      await act(async () => {
        result.current.clipboard.paste({ x: 0, y: 0 });
        await new Promise((r) => setTimeout(r, 20));
      });
      expect(result.current.nodes).toHaveLength(4);
    });

    it("paste is a no-op when navigator.clipboard.read rejects", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      readMock.mockRejectedValueOnce(new Error("permission denied"));

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      await act(async () => {
        result.current.clipboard.paste({ x: 0, y: 0 });
        await new Promise((r) => setTimeout(r, 20));
      });
      expect(result.current.nodes).toHaveLength(4);
    });

    it("paste ignores a MIME payload with a mismatched version", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      readResult = [
        buildReadItem({
          version: 999,
          nodes: [],
          edges: [],
          configs: {},
          anchor: { x: 0, y: 0 },
        }),
      ];

      await act(async () => {
        result.current.clipboard.paste({ x: 0, y: 0 });
        await new Promise((r) => setTimeout(r, 20));
      });
      expect(result.current.nodes).toHaveLength(4);
    });

    it("copy then paste roundtrips with fresh keys at the cursor offset", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createSchematicWithGroup(Wrapper);

      const { result } = renderHook(
        () => ({
          clipboard: Schematic.useClipboard({ key: schem.key, selected: ["g1"] }),
          nodes: Schematic.useSelectAllNodes({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );

      act(() => result.current.clipboard.copy());
      await waitFor(() => expect(writeMock).toHaveBeenCalledTimes(1));
      readResult = writeMock.mock.calls[0][0] as FakeClipboardItem[];

      act(() => result.current.clipboard.paste({ x: 500, y: 500 }));
      await waitFor(() => expect(result.current.nodes).toHaveLength(7));
    });
  });
});
