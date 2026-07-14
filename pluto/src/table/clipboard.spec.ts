// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, table } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import {
  type ClipboardEvent as ReactClipboardEvent,
  type PropsWithChildren,
} from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Table } from "@/table";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

// makeClipboardEvent fakes the minimum DataTransfer surface useClipboard
// reads/writes. preventDefault is captured so tests can assert handlers
// actually claimed the event.
const makeClipboardEvent = (
  initialData: Record<string, string> = {},
): ReactClipboardEvent<HTMLElement> & { _data: Record<string, string> } => {
  const data: Record<string, string> = { ...initialData };
  let prevented = false;
  return {
    preventDefault: () => {
      prevented = true;
    },
    get defaultPrevented() {
      return prevented;
    },
    clipboardData: {
      setData: (type: string, value: string) => {
        data[type] = value;
      },
      getData: (type: string) => data[type] ?? "",
    },
    _data: data,
  } as unknown as ReactClipboardEvent<HTMLElement> & {
    _data: Record<string, string>;
  };
};

const MIME = "web application/synnax-table+json";

const valueCfg = (units: string): table.CellConfig =>
  table.cellConfigZ.parse({ variant: "value", units });
const textCfg = (value: string): table.CellConfig =>
  table.cellConfigZ.parse({ variant: "text", value });

describe("table clipboard", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createTable = async () => {
    const proj = await client.projects.create({
      name: `clipboard_ws_${uuid.create()}`,
      layout: {},
    });
    return await client.tables.create(proj.key, {
      name: "clipboard_test",
      rows: [
        { size: 30, cells: ["a", "b", "c"] },
        { size: 30, cells: ["d", "e", "f"] },
        { size: 30, cells: ["g", "h", "i"] },
      ],
      columns: [{ size: 80 }, { size: 80 }, { size: 80 }],
      cells: {
        a: valueCfg("A"),
        b: valueCfg("B"),
        c: valueCfg("C"),
        d: textCfg("D"),
        e: textCfg("E"),
        f: textCfg("F"),
        g: valueCfg("G"),
        h: valueCfg("H"),
        i: valueCfg("I"),
      },
    });
  };

  const loadAndUse = async (key: table.Key, selected: string[]) => {
    const retrieve = renderHook(() => Table.useRetrieve({ key }), { wrapper });
    await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
    return renderHook(
      () => ({
        retrieve: Table.useRetrieve({ key }),
        clipboard: Table.useClipboard({ key, selected }),
      }),
      { wrapper },
    );
  };

  describe("copy", () => {
    it("writes a payload to clipboardData normalized to the top-left of the selection", async () => {
      const created = await createTable();
      const { result } = await loadAndUse(created.key, ["e", "f", "h", "i"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const ev = makeClipboardEvent();
      act(() => result.current.clipboard.onCopy(ev));
      expect(ev.defaultPrevented).toBe(true);
      const payload = JSON.parse(ev._data[MIME]) as {
        version: number;
        cells: Array<{ row: number; col: number; config: table.CellConfig }>;
      };
      expect(payload.version).toEqual(1);
      const positions = new Map(
        payload.cells.map((c) => [`${c.row},${c.col}`, c.config.variant]),
      );
      expect(positions.get("0,0")).toEqual("text");
      expect(positions.get("0,1")).toEqual("text");
      expect(positions.get("1,0")).toEqual("value");
      expect(positions.get("1,1")).toEqual("value");
    });

    it("writes a human-readable text/plain summary", async () => {
      const created = await createTable();
      const { result } = await loadAndUse(created.key, ["a", "b"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const ev = makeClipboardEvent();
      act(() => result.current.clipboard.onCopy(ev));
      expect(ev._data["text/plain"]).toEqual("2 cells");
    });

    it("is a no-op when selection is empty", async () => {
      const created = await createTable();
      const { result } = await loadAndUse(created.key, []);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const ev = makeClipboardEvent();
      act(() => result.current.clipboard.onCopy(ev));
      expect(ev.defaultPrevented).toBe(false);
      expect(ev._data[MIME]).toBeUndefined();
    });
  });

  describe("paste", () => {
    it("overwrites in-bounds cells at the anchor position", async () => {
      const created = await createTable();
      const { result } = await loadAndUse(created.key, ["a"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 1,
        cells: [
          { row: 0, col: 0, config: valueCfg("X") },
          { row: 0, col: 1, config: valueCfg("Y") },
        ],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      await waitFor(() => {
        expect(result.current.retrieve.data?.cells.a).toMatchObject(valueCfg("X"));
        expect(result.current.retrieve.data?.cells.b).toMatchObject(valueCfg("Y"));
      });
      expect(result.current.retrieve.data?.rows).toHaveLength(3);
      expect(result.current.retrieve.data?.columns).toHaveLength(3);
    });

    it("grows the table when the paste extends past the right edge", async () => {
      const created = await createTable();
      const { result } = await loadAndUse(created.key, ["c"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 1,
        cells: [
          { row: 0, col: 0, config: textCfg("P") },
          { row: 0, col: 1, config: textCfg("Q") },
        ],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      await waitFor(() => {
        expect(result.current.retrieve.data?.columns).toHaveLength(4);
        expect(result.current.retrieve.data?.rows[0].cells).toHaveLength(4);
        expect(result.current.retrieve.data?.cells.c).toMatchObject(textCfg("P"));
      });
    });

    it("grows the table when the paste extends past the bottom edge", async () => {
      const created = await createTable();
      const { result } = await loadAndUse(created.key, ["g"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 1,
        cells: [
          { row: 0, col: 0, config: textCfg("P") },
          { row: 1, col: 0, config: textCfg("Q") },
        ],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      await waitFor(() => {
        expect(result.current.retrieve.data?.rows).toHaveLength(4);
        expect(result.current.retrieve.data?.cells.g).toMatchObject(textCfg("P"));
      });
    });

    it("grows both axes when the paste extends past the bottom-right corner", async () => {
      const created = await createTable();
      const { result } = await loadAndUse(created.key, ["i"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 1,
        cells: [
          { row: 0, col: 0, config: valueCfg("X") },
          { row: 1, col: 1, config: valueCfg("Y") },
        ],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      await waitFor(() => {
        expect(result.current.retrieve.data?.rows).toHaveLength(4);
        expect(result.current.retrieve.data?.columns).toHaveLength(4);
      });
    });

    it("is a no-op when selection is empty", async () => {
      const created = await createTable();
      const { result } = await loadAndUse(created.key, []);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 1,
        cells: [{ row: 0, col: 0, config: valueCfg("X") }],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      expect(ev.defaultPrevented).toBe(false);
      expect(result.current.retrieve.data?.cells.a).toMatchObject(valueCfg("A"));
    });

    it("is a no-op when the payload version is unknown", async () => {
      const created = await createTable();
      const { result } = await loadAndUse(created.key, ["a"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 99,
        cells: [{ row: 0, col: 0, config: valueCfg("X") }],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      expect(ev.defaultPrevented).toBe(false);
      expect(result.current.retrieve.data?.cells.a).toMatchObject(valueCfg("A"));
    });

    it("fires onPaste with the keys that were overwritten", async () => {
      const created = await createTable();
      const retrieve = renderHook(() => Table.useRetrieve({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
      const seen: string[][] = [];
      const { result } = renderHook(
        () =>
          Table.useClipboard({
            key: created.key,
            selected: ["a"],
            onPaste: (keys) => seen.push(keys),
          }),
        { wrapper },
      );
      const payload = {
        version: 1,
        cells: [
          { row: 0, col: 0, config: valueCfg("X") },
          { row: 0, col: 1, config: valueCfg("Y") },
        ],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.onPaste(ev));
      await waitFor(() => expect(seen.length).toBeGreaterThan(0));
      expect(seen[0]).toEqual(["a", "b"]);
    });
  });

  describe("round-trip", () => {
    it("copy then paste at a new anchor reproduces the source region", async () => {
      const created = await createTable();
      const retrieve = renderHook(() => Table.useRetrieve({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));

      const copyHook = renderHook(
        () => Table.useClipboard({ key: created.key, selected: ["a", "b", "d", "e"] }),
        { wrapper },
      );
      const copyEv = makeClipboardEvent();
      act(() => copyHook.result.current.onCopy(copyEv));
      const wire = copyEv._data[MIME];
      expect(wire.length).toBeGreaterThan(0);

      const pasteHook = renderHook(
        () => Table.useClipboard({ key: created.key, selected: ["e"] }),
        { wrapper },
      );
      const pasteEv = makeClipboardEvent({ [MIME]: wire });
      await act(async () => pasteHook.result.current.onPaste(pasteEv));
      await waitFor(() => {
        expect(retrieve.result.current.data?.cells.e).toMatchObject(valueCfg("A"));
        expect(retrieve.result.current.data?.cells.f).toMatchObject(valueCfg("B"));
        expect(retrieve.result.current.data?.cells.h).toMatchObject(textCfg("D"));
        expect(retrieve.result.current.data?.cells.i).toMatchObject(textCfg("E"));
      });
    });
  });
});
