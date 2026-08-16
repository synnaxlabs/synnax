// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Table } from "@/feature/table";
import {
  client,
  createCellGrid,
  createPreloadedState,
  renderTable,
} from "@/feature/table/testutil";
import { Session } from "@/session";
import { getLabeledDialogTrigger, uniqueName } from "@/testutil";

interface RenderToolbarOptions {
  tableState?: Partial<Session.Table.State>;
  withCells?: boolean;
}

const renderToolbar = async ({
  tableState,
  withCells = true,
}: RenderToolbarOptions = {}) => {
  const name = uniqueName("table");
  const handle = await renderTable(Table.Toolbar, {
    table: { name, ...(withCells ? createCellGrid() : {}) },
    preloadedState: (key) => createPreloadedState(key, tableState),
  });
  return { name, ...handle };
};

describe("table/Toolbar", () => {
  it("displays the table name in the header", async () => {
    const { name } = await renderToolbar();
    expect(await screen.findByText(name)).toBeDefined();
  });

  it("prompts to enable editing when the table is not editable", async () => {
    const { name, key, store } = await renderToolbar({
      tableState: { editable: false },
    });
    expect(
      await screen.findByText(new RegExp(`${name} is not editable`)),
    ).toBeDefined();
    fireEvent.click(await screen.findByText("enable editing."));
    await waitFor(() =>
      expect(
        Session.Table.selectSliceState(store.getState()).tables[key].editable,
      ).toBe(true),
    );
  });

  it("shows the empty state when no cell is selected", async () => {
    await renderToolbar();
    expect(
      await screen.findByText(
        "No cell selected. Select a cell to view its properties.",
      ),
    ).toBeDefined();
  });

  it("shows the cell position and form for a single selected cell", async () => {
    const { result } = await renderToolbar({
      tableState: { selectedCells: ["a"], lastSelected: "a" },
    });
    expect(await screen.findByDisplayValue("Cell A")).toBeDefined();
    await waitFor(() => expect(result.container.textContent).toContain("A1"));
  });

  it("persists a text cell edit to the server", async () => {
    const { key } = await renderToolbar({
      tableState: { selectedCells: ["a"], lastSelected: "a" },
    });
    const input = await screen.findByDisplayValue("Cell A");
    fireEvent.change(input, { target: { value: "Updated" } });
    await waitFor(async () => {
      const t = await client.tables.retrieve(key);
      expect(t.cells.a.props.value).toBe("Updated");
    });
  });

  it("swaps the cell variant while preserving compatible props", async () => {
    const { key } = await renderToolbar({
      tableState: { selectedCells: ["a"], lastSelected: "a" },
    });
    await screen.findByDisplayValue("Cell A");
    fireEvent.click(getLabeledDialogTrigger("Variant"));
    fireEvent.click(await screen.findByText("Value"));
    await waitFor(async () => {
      const t = await client.tables.retrieve(key);
      expect(t.cells.a.variant).toBe("value");
    });
  });

  it("shows the multi-cell form with the shared cell count", async () => {
    const { result } = await renderToolbar({
      tableState: { selectedCells: ["a", "b"], lastSelected: "b" },
    });
    expect(await screen.findByText("Variant")).toBeDefined();
    expect(screen.getByText("Selection colors")).toBeDefined();
    expect(screen.getByText("Size")).toBeDefined();
    await waitFor(() => expect(result.container.textContent).toContain("2 cells"));
  });

  it("applies a size change to every selected cell", async () => {
    const { key } = await renderToolbar({
      tableState: { selectedCells: ["a", "b"], lastSelected: "b" },
    });
    await screen.findByText("Size");
    fireEvent.click(screen.getByText("M"));
    await waitFor(async () => {
      const t = await client.tables.retrieve(key);
      expect(t.cells.a.props.level).toBe("h4");
      expect(t.cells.b.props.level).toBe("h4");
    });
  });
});
