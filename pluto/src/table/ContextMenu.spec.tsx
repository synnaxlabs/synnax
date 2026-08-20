// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type table } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Table } from "@/table";
import {
  DefaultContextMenu,
  type DefaultContextMenuProps,
  ERASE_TRIGGER,
} from "@/table/ContextMenu";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

// Single-hook bootstrap so the suspending useEnsure is not followed by other
// hooks, which trips a React 19 concurrent replay warning (same pattern as
// table queries.spec.tsx).
const loadTable = async (
  wrapper: React.FC<PropsWithChildren>,
  key: table.Key,
): Promise<void> => {
  const Bootstrap = (): ReactElement => {
    Table.useEnsure({ key });
    return <div data-testid="loaded" />;
  };
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(
      <Errors.SuspenseBoundary loading={null}>
        <Bootstrap />
      </Errors.SuspenseBoundary>,
      { wrapper },
    );
  });
  await utils.findByTestId("loaded");
};

describe("table DefaultContextMenu", () => {
  let wrapper: React.FC<PropsWithChildren>;
  let key: table.Key;

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    const project = await client.projects.create({ name: "menu_project", layout: {} });
    const created = await client.tables.create(project.key, {
      name: "menu_table",
      rows: [{ size: 36, cells: ["a"] }],
      columns: [{ size: 72 }],
      cells: { a: { key: "a", variant: "text", props: { value: "A" } } },
    });
    key = created.key;
    await loadTable(wrapper, key);
  });

  const renderMenu = (props: Partial<DefaultContextMenuProps> = {}) =>
    render(
      <DefaultContextMenu
        resourceKey={key}
        targetID={null}
        selected={[]}
        editable={false}
        onAddRow={vi.fn()}
        onAddCol={vi.fn()}
        onRemoveRow={vi.fn()}
        onRemoveCol={vi.fn()}
        onEraseCells={vi.fn()}
        {...props}
      />,
      { wrapper },
    );

  it("offers undo and redo to an editor", async () => {
    renderMenu({ editable: true });
    expect(await screen.findByText("Undo")).toBeDefined();
    expect(screen.getByText("Redo")).toBeDefined();
  });

  it("withholds undo and redo from a read-only table", async () => {
    renderMenu({ editable: false, onCenteredChange: vi.fn() });
    // Positive control first: the menu really rendered, so the absence below is
    // the read-only gate rather than an empty menu.
    expect(await screen.findByText("Center table")).toBeDefined();
    expect(screen.queryByText("Undo")).toBeNull();
    expect(screen.queryByText("Redo")).toBeNull();
  });

  it("advertises the erase shortcut the table binds", async () => {
    renderMenu({ editable: true, selected: ["a"] });
    const item = (await screen.findByText("Erase cell")).closest("button");
    const hint = item?.querySelector('[aria-label="trigger-indicator"]');
    // The item never binds the key itself, so a hint that drifted from the trigger
    // Table registers would advertise a key that does nothing. Keycaps follow the
    // host platform's casing, so the comparison ignores it.
    expect(hint?.textContent?.toLowerCase()).toContain(ERASE_TRIGGER[0].toLowerCase());
  });

  it("omits the centering item when onCenteredChange is not provided", () => {
    renderMenu();
    expect(screen.queryByText("Center table")).toBeNull();
    expect(screen.queryByText("Align table to top left")).toBeNull();
  });

  it("centers an uncentered table", async () => {
    const onCenteredChange = vi.fn();
    renderMenu({ centered: false, onCenteredChange });
    fireEvent.click(await screen.findByText("Center table"));
    expect(onCenteredChange).toHaveBeenCalledWith(true);
  });

  it("returns a centered table to the top left", async () => {
    const onCenteredChange = vi.fn();
    renderMenu({ centered: true, onCenteredChange });
    fireEvent.click(await screen.findByText("Align table to top left"));
    expect(onCenteredChange).toHaveBeenCalledWith(false);
  });

  // Centering applies in both modes, unlike the indicators item, which hides
  // while editing because indicators are forced visible.
  it("offers centering while editing", async () => {
    renderMenu({
      editable: true,
      onCenteredChange: vi.fn(),
      onShowIndicatorsChange: vi.fn(),
    });
    expect(await screen.findByText("Center table")).toBeDefined();
    expect(screen.queryByText("Hide indicators")).toBeNull();
  });
});
