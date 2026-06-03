// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type table } from "@synnaxlabs/client";
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

describe("table clipboard", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const seedTable = async () => {
    const ws = await client.projects.create({
      name: `clipboard_ws_${uuid.create()}`,
    });
    return await client.tables.create(ws.key, {
      name: "clipboard_test",
      rows: [
        { size: 30, cells: ["a", "b", "c"] },
        { size: 30, cells: ["d", "e", "f"] },
        { size: 30, cells: ["g", "h", "i"] },
      ],
      columns: [{ size: 80 }, { size: 80 }, { size: 80 }],
      cells: {
        a: { key: "a", variant: "value", props: { v: "A" } },
        b: { key: "b", variant: "value", props: { v: "B" } },
        c: { key: "c", variant: "value", props: { v: "C" } },
        d: { key: "d", variant: "text", props: { v: "D" } },
        e: { key: "e", variant: "text", props: { v: "E" } },
        f: { key: "f", variant: "text", props: { v: "F" } },
        g: { key: "g", variant: "value", props: { v: "G" } },
        h: { key: "h", variant: "value", props: { v: "H" } },
        i: { key: "i", variant: "value", props: { v: "I" } },
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
      const seeded = await seedTable();
      const { result } = await loadAndUse(seeded.key, ["e", "f", "h", "i"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const ev = makeClipboardEvent();
      act(() => result.current.clipboard.onCopy(ev));
      expect(ev.defaultPrevented).toBe(true);
      const payload = JSON.parse(ev._data[MIME]) as {
        version: number;
        cells: Array<{ row: number; col: number; variant: string }>;
      };
      expect(payload.version).toEqual(1);
      const positions = new Map(
        payload.cells.map((c) => [`${c.row},${c.col}`, c.variant]),
      );
      expect(positions.get("0,0")).toEqual("text");
      expect(positions.get("0,1")).toEqual("text");
      expect(positions.get("1,0")).toEqual("value");
      expect(positions.get("1,1")).toEqual("value");
    });

    it("writes a human-readable text/plain summary", async () => {
      const seeded = await seedTable();
      const { result } = await loadAndUse(seeded.key, ["a", "b"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const ev = makeClipboardEvent();
      act(() => result.current.clipboard.onCopy(ev));
      expect(ev._data["text/plain"]).toEqual("2 cells");
    });

    it("is a no-op when selection is empty", async () => {
      const seeded = await seedTable();
      const { result } = await loadAndUse(seeded.key, []);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const ev = makeClipboardEvent();
      act(() => result.current.clipboard.onCopy(ev));
      expect(ev.defaultPrevented).toBe(false);
      expect(ev._data[MIME]).toBeUndefined();
    });
  });

  describe("paste", () => {
    it("overwrites in-bounds cells at the anchor position", async () => {
      const seeded = await seedTable();
      const { result } = await loadAndUse(seeded.key, ["a"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 1,
        cells: [
          { row: 0, col: 0, variant: "value", props: { v: "X" } },
          { row: 0, col: 1, variant: "value", props: { v: "Y" } },
        ],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      await waitFor(() => {
        expect(result.current.retrieve.data?.cells.a.props.v).toEqual("X");
        expect(result.current.retrieve.data?.cells.b.props.v).toEqual("Y");
      });
      expect(result.current.retrieve.data?.rows).toHaveLength(3);
      expect(result.current.retrieve.data?.columns).toHaveLength(3);
    });

    it("grows the table when the paste extends past the right edge", async () => {
      const seeded = await seedTable();
      const { result } = await loadAndUse(seeded.key, ["c"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 1,
        cells: [
          { row: 0, col: 0, variant: "text", props: { v: "P" } },
          { row: 0, col: 1, variant: "text", props: { v: "Q" } },
        ],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      await waitFor(() => {
        expect(result.current.retrieve.data?.columns).toHaveLength(4);
        expect(result.current.retrieve.data?.rows[0].cells).toHaveLength(4);
        expect(result.current.retrieve.data?.cells.c.props.v).toEqual("P");
      });
    });

    it("grows the table when the paste extends past the bottom edge", async () => {
      const seeded = await seedTable();
      const { result } = await loadAndUse(seeded.key, ["g"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 1,
        cells: [
          { row: 0, col: 0, variant: "text", props: { v: "P" } },
          { row: 1, col: 0, variant: "text", props: { v: "Q" } },
        ],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      await waitFor(() => {
        expect(result.current.retrieve.data?.rows).toHaveLength(4);
        expect(result.current.retrieve.data?.cells.g.props.v).toEqual("P");
      });
    });

    it("grows both axes when the paste extends past the bottom-right corner", async () => {
      const seeded = await seedTable();
      const { result } = await loadAndUse(seeded.key, ["i"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 1,
        cells: [
          { row: 0, col: 0, variant: "value", props: { v: "X" } },
          { row: 1, col: 1, variant: "value", props: { v: "Y" } },
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
      const seeded = await seedTable();
      const { result } = await loadAndUse(seeded.key, []);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 1,
        cells: [{ row: 0, col: 0, variant: "value", props: { v: "X" } }],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      expect(ev.defaultPrevented).toBe(false);
      expect(result.current.retrieve.data?.cells.a.props.v).toEqual("A");
    });

    it("is a no-op when the payload version is unknown", async () => {
      const seeded = await seedTable();
      const { result } = await loadAndUse(seeded.key, ["a"]);
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      const payload = {
        version: 99,
        cells: [{ row: 0, col: 0, variant: "value", props: { v: "X" } }],
      };
      const ev = makeClipboardEvent({ [MIME]: JSON.stringify(payload) });
      await act(async () => result.current.clipboard.onPaste(ev));
      expect(ev.defaultPrevented).toBe(false);
      expect(result.current.retrieve.data?.cells.a.props.v).toEqual("A");
    });

    it("fires onPaste with the keys that were overwritten", async () => {
      const seeded = await seedTable();
      const retrieve = renderHook(() => Table.useRetrieve({ key: seeded.key }), {
        wrapper,
      });
      await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
      const seen: string[][] = [];
      const { result } = renderHook(
        () =>
          Table.useClipboard({
            key: seeded.key,
            selected: ["a"],
            onPaste: (keys) => seen.push(keys),
          }),
        { wrapper },
      );
      const payload = {
        version: 1,
        cells: [
          { row: 0, col: 0, variant: "value", props: { v: "X" } },
          { row: 0, col: 1, variant: "value", props: { v: "Y" } },
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
      const seeded = await seedTable();
      const retrieve = renderHook(() => Table.useRetrieve({ key: seeded.key }), {
        wrapper,
      });
      await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));

      const copyHook = renderHook(
        () => Table.useClipboard({ key: seeded.key, selected: ["a", "b", "d", "e"] }),
        { wrapper },
      );
      const copyEv = makeClipboardEvent();
      act(() => copyHook.result.current.onCopy(copyEv));
      const wire = copyEv._data[MIME];
      expect(wire.length).toBeGreaterThan(0);

      const pasteHook = renderHook(
        () => Table.useClipboard({ key: seeded.key, selected: ["e"] }),
        { wrapper },
      );
      const pasteEv = makeClipboardEvent({ [MIME]: wire });
      await act(async () => pasteHook.result.current.onPaste(pasteEv));
      await waitFor(() => {
        expect(retrieve.result.current.data?.cells.e.variant).toEqual("value");
        expect(retrieve.result.current.data?.cells.e.props.v).toEqual("A");
        expect(retrieve.result.current.data?.cells.f.variant).toEqual("value");
        expect(retrieve.result.current.data?.cells.f.props.v).toEqual("B");
        expect(retrieve.result.current.data?.cells.h.variant).toEqual("text");
        expect(retrieve.result.current.data?.cells.h.props.v).toEqual("D");
        expect(retrieve.result.current.data?.cells.i.variant).toEqual("text");
        expect(retrieve.result.current.data?.cells.i.props.v).toEqual("E");
      });
    });
  });
});
