// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { color } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  client,
  createCellGrid,
  createPreloadedState,
  renderTable,
} from "@/feature/table/testutil";
import { Toolbar } from "@/feature/table/Toolbar";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import { assertDefined, getInputByItemLabel, uniqueName } from "@/testutil";

interface RenderToolbarOptions {
  tableState?: Partial<Session.Table.State>;
  withCells?: boolean;
  as?: Client;
}

const roles = new RoleClients(client);

const createUncoloredGrid = (): ReturnType<typeof createCellGrid> => ({
  ...createCellGrid(),
  cells: {
    a: { variant: "text", value: "Cell A", level: "h5" },
    b: { variant: "text", value: "Cell B", level: "h5" },
  },
});

// The role control's hex box, labeled with the role name.
const roleInput = (label: string): HTMLInputElement => {
  const input = screen
    .getAllByLabelText(label)
    .find((el): el is HTMLInputElement => el instanceof HTMLInputElement);
  assertDefined(input);
  return input;
};

const hexOf = (value: unknown): string | undefined =>
  value == null ? undefined : color.hex(value as color.Crude);

const renderToolbar = async ({
  tableState,
  withCells = true,
  as,
}: RenderToolbarOptions = {}) => {
  const name = uniqueName("table");
  const handle = await renderTable(Toolbar, {
    table: { name, ...(withCells ? createCellGrid() : {}) },
    preloadedState: (key) => createPreloadedState(key, tableState),
    as,
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
    fireEvent.click(await screen.findByText("Enable editing"));
    await waitFor(() =>
      expect(
        documentIn(Session.Table.selectSliceState(store.getState()), key)?.editable,
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
      expect(t.cells.a).toMatchObject({ variant: "text", value: "Updated" });
    });
  });

  it("swaps the cell variant while preserving compatible fields", async () => {
    const { key } = await renderToolbar({
      tableState: { selectedCells: ["a"], lastSelected: "a" },
    });
    await screen.findByDisplayValue("Cell A");
    fireEvent.click(screen.getByLabelText("Change cell type"));
    fireEvent.click(await screen.findByText("Value"));
    await waitFor(async () => {
      const t = await client.tables.retrieve(key);
      expect(t.cells.a.variant).toBe("value");
    });
  });

  it("swaps the variant of every selected cell from the header", async () => {
    const { key } = await renderToolbar({
      tableState: { selectedCells: ["a", "b"], lastSelected: "b" },
    });
    await screen.findByText("Selection");
    fireEvent.click(screen.getByLabelText("Change cell type"));
    fireEvent.click(await screen.findByText("Value"));
    await waitFor(async () => {
      const t = await client.tables.retrieve(key);
      expect([t.cells.a.variant, t.cells.b.variant]).toEqual(["value", "value"]);
    });
  });

  it("shows the multi-cell form with the shared cell count", async () => {
    const { result } = await renderToolbar({
      tableState: { selectedCells: ["a", "b"], lastSelected: "b" },
    });
    expect(await screen.findByText("Selection")).toBeDefined();
    expect(screen.getByText("Size")).toBeDefined();
    await waitFor(() => expect(result.container.textContent).toContain("2 cells"));
  });

  it("keeps the selected form tab when another value cell is selected", async () => {
    const { key, store } = await renderTable(Toolbar, {
      table: {
        name: uniqueName("table"),
        rows: [{ size: 36, cells: ["a", "b"] }],
        columns: [{ size: 72 }, { size: 72 }],
        cells: { a: { variant: "value" }, b: { variant: "value" } },
      },
      preloadedState: (key) =>
        createPreloadedState(key, { selectedCells: ["a"], lastSelected: "a" }),
    });
    const telemetry = await screen.findByRole("tab", { name: "Telemetry" });
    expect(telemetry.ariaSelected).toBe("true");
    fireEvent.click(screen.getByRole("tab", { name: "Style" }));
    act(() => {
      store.dispatch(
        Session.Table.setSelectedCells({ key, cells: ["b"], anchor: "b" }),
      );
    });
    await waitFor(() =>
      expect(screen.getByRole("tab", { name: "Style" }).ariaSelected).toBe("true"),
    );
  });

  it("sets a fill on every selected cell whose fill is auto", async () => {
    const { key } = await renderTable(Toolbar, {
      table: { name: uniqueName("table"), ...createUncoloredGrid() },
      preloadedState: (key) =>
        createPreloadedState(key, { selectedCells: ["a", "b"], lastSelected: "b" }),
    });
    await screen.findByText("Colors");
    expect(roleInput("Fill").placeholder).toBe("Auto");
    expect(screen.queryByText("Selection")).toBeNull();
    fireEvent.change(roleInput("Fill"), { target: { value: "0000ff" } });
    await waitFor(async () => {
      const t = await client.tables.retrieve(key);
      expect([t.cells.a, t.cells.b].map((c) => hexOf(c.fillColor))).toEqual([
        "#0000ff",
        "#0000ff",
      ]);
    });
  });

  it("shows Mixed when the selected fills differ", async () => {
    await renderToolbar({
      tableState: { selectedCells: ["a", "b"], lastSelected: "b" },
    });
    await screen.findByText("Colors");
    expect(roleInput("Fill").placeholder).toBe("Mixed");
    expect(roleInput("Text").placeholder).toBe("Auto");
  });

  it("recolors only the cells that hold a selection color", async () => {
    const { key } = await renderToolbar({
      tableState: { selectedCells: ["a", "b"], lastSelected: "b" },
    });
    const label = await screen.findByText("Selection");
    const item = label.closest(".pluto-input__item");
    assertDefined(item);
    const swatches = item.querySelectorAll<HTMLElement>(".pluto-color-swatch");
    expect(swatches).toHaveLength(2);
    fireEvent.click(swatches[0]);
    fireEvent.change(screen.getByLabelText("Hex"), { target: { value: "0000ff" } });
    fireEvent.keyDown(document.body, { code: "Escape" });
    await waitFor(async () => {
      const t = await client.tables.retrieve(key);
      expect([t.cells.a, t.cells.b].map((c) => hexOf(c.fillColor))).toEqual([
        "#0000ff",
        "#00ff00",
      ]);
    });
  });

  it("writes a number format only to the value cells", async () => {
    const { key, result } = await renderTable(Toolbar, {
      table: {
        name: uniqueName("table"),
        rows: [{ size: 36, cells: ["a", "b"] }],
        columns: [{ size: 72 }, { size: 72 }],
        cells: { a: { variant: "text", value: "Cell A" }, b: { variant: "value" } },
      },
      preloadedState: (key) =>
        createPreloadedState(key, { selectedCells: ["a", "b"], lastSelected: "b" }),
    });
    await screen.findByText("Number format");
    expect(screen.getByText("Staleness")).toBeDefined();
    const input = getInputByItemLabel(result.container, "Precision");
    fireEvent.change(input, { target: { value: "4" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const t = await client.tables.retrieve(key);
      expect(t.cells.b).toMatchObject({ precision: 4 });
    });
    const t = await client.tables.retrieve(key);
    expect(t.cells.a).not.toHaveProperty("precision");
  });

  it("hides the value groups when only text cells are selected", async () => {
    await renderToolbar({
      tableState: { selectedCells: ["a", "b"], lastSelected: "b" },
    });
    await screen.findByText("Colors");
    expect(screen.queryByText("Number format")).toBeNull();
    expect(screen.queryByText("Staleness")).toBeNull();
  });

  it("applies a size change to every selected cell", async () => {
    const { key } = await renderToolbar({
      tableState: { selectedCells: ["a", "b"], lastSelected: "b" },
    });
    await screen.findByText("Size");
    fireEvent.click(screen.getByText("M"));
    await waitFor(async () => {
      const t = await client.tables.retrieve(key);
      expect(t.cells.a).toMatchObject({ level: "h4" });
      expect(t.cells.b).toMatchObject({ level: "h4" });
    });
  });
});

describe("table/Toolbar permissions", () => {
  it("should not invite a viewer to enable editing", async () => {
    const { name } = await renderToolbar({
      tableState: { editable: false },
      as: await roles.get("Viewer"),
    });
    expect(await screen.findByText(`${name} is not editable`)).toBeTruthy();
    expect(screen.queryByText("enable editing.")).toBeNull();
  });
});
