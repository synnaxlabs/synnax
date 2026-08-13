// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Table } from "@/table";
import { table as aetherTable } from "@/table/aether";
import { mockBoundingClientRect } from "@/testutil/dom";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/test";

const client = createTestClient();

const SURFACE_WIDTH = 1000;
const SURFACE_HEIGHT = 600;
const COL_SIZE = 72;
const ROW_SIZE = 36;
// Indicator strips occupy one row and one column outside the stored sizes.
const INDICATOR_SIZE = 4.5 * 6;

const expectedOffset = (contentWidth: number, contentHeight: number) => ({
  x: Math.floor((SURFACE_WIDTH - contentWidth - INDICATOR_SIZE) / 2),
  y: Math.floor((SURFACE_HEIGHT - contentHeight - INDICATOR_SIZE) / 2),
});

const RECT = mockBoundingClientRect(0, 0, SURFACE_WIDTH, SURFACE_HEIGHT)();

// The @juggle polyfill reads computed styles, which jsdom leaves empty, so it
// never reports a size. useResize measures with getBoundingClientRect and
// ignores the entries, so notifying is enough.
class ImmediateResizeObserver {
  constructor(private readonly notify: () => void) {}
  observe(): void {
    this.notify();
  }
  unobserve(): void {}
  disconnect(): void {}
}

describe("Table centering", () => {
  let wrapper: React.FC<PropsWithChildren>;
  let key: table.Key;

  beforeEach(async () => {
    vi.stubGlobal("ResizeObserver", ImmediateResizeObserver);
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(RECT);
    // Canvas.useRegion measures against the lower2d canvas and bails without it.
    const canvas = document.createElement("div");
    canvas.className = "pluto-canvas--lower2d";
    document.body.appendChild(canvas);
    wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: aetherTable.REGISTRY,
      renderContext: canvasTest.record(),
    });
    const project = await client.projects.create({ name: "center", layout: {} });
    const created = await client.tables.create(project.key, {
      name: "center_table",
      rows: [{ size: ROW_SIZE, cells: ["a"] }],
      columns: [{ size: COL_SIZE }],
      cells: { a: { key: "a", variant: "text", props: { value: "A" } } },
    });
    key = created.key;
    const retrieve = renderHook(() => Table.useRetrieve({ key }), { wrapper });
    await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
  });

  afterEach(() => {
    vi.restoreAllMocks();
    document.querySelector(".pluto-canvas--lower2d")?.remove();
  });

  const renderTable = (centered: boolean) => {
    const Wrapped = (): ReactElement => (
      <Table.Suspended tableKey={key}>
        <Table.Table centered={centered} visible />
      </Table.Suspended>
    );
    const { container } = render(<Wrapped />, { wrapper });
    const frame = (): HTMLElement => {
      const el = container.querySelector<HTMLElement>(".pluto-table-frame");
      if (el == null) throw new Error("the table frame did not render");
      return el;
    };
    return { frame };
  };

  const resizeRow = async (size: number): Promise<void> => {
    const { result } = renderHook(() => Table.useDispatch(), { wrapper });
    await act(async () => {
      await result.current.dispatchAsync({
        key,
        actions: [table.resizeRow({ index: 0, size })],
      });
    });
  };

  const expectTransform = async (
    frame: () => HTMLElement,
    offset: { x: number; y: number },
  ) =>
    await waitFor(() =>
      expect(frame().style.transform).toEqual(
        `translate(${offset.x}px, ${offset.y}px)`,
      ),
    );

  it("leaves an uncentered table untranslated", async () => {
    const { frame } = renderTable(false);
    await waitFor(() => expect(frame().style.transform).toEqual(""));
  });

  it("centers on both axes when the table fits", async () => {
    const { frame } = renderTable(true);
    await expectTransform(frame, expectedOffset(COL_SIZE, ROW_SIZE));
  });

  it("clamps to zero on an axis the table overflows", async () => {
    const { frame } = renderTable(true);
    await resizeRow(SURFACE_HEIGHT * 2);
    await expectTransform(frame, {
      ...expectedOffset(COL_SIZE, SURFACE_HEIGHT * 2),
      y: 0,
    });
  });

  it("recenters when the table changes size", async () => {
    const { frame } = renderTable(true);
    await expectTransform(frame, expectedOffset(COL_SIZE, ROW_SIZE));
    await resizeRow(200);
    await expectTransform(frame, expectedOffset(COL_SIZE, 200));
  });

  it("holds the offset while the pointer is down", async () => {
    const { frame } = renderTable(true);
    const settled = expectedOffset(COL_SIZE, ROW_SIZE);
    await expectTransform(frame, settled);
    fireEvent.pointerDown(frame());
    await resizeRow(200);
    expect(frame().style.transform).toEqual(
      `translate(${settled.x}px, ${settled.y}px)`,
    );
    fireEvent.pointerUp(window);
    await expectTransform(frame, expectedOffset(COL_SIZE, 200));
  });

  it("releases the held offset on pointer cancellation", async () => {
    const { frame } = renderTable(true);
    await expectTransform(frame, expectedOffset(COL_SIZE, ROW_SIZE));
    fireEvent.pointerDown(frame());
    await resizeRow(200);
    fireEvent.pointerCancel(window);
    await expectTransform(frame, expectedOffset(COL_SIZE, 200));
  });
});
