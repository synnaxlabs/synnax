// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import {
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  getIconButton,
  type TestStore,
  uniqueName,
  waitForPlacedLayout,
} from "@/testutil";

const client = createTestClient();

const createArc = async () =>
  await client.arcs.create({
    name: uniqueName("arc"),
    mode: "graph",
    graph: { nodes: [], edges: [] },
  });

const renderToolbar = async (): Promise<{ store: TestStore }> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  render(
    <>
      {Arc.TOOLBAR.content}
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { store };
};

describe("arc/Toolbar", () => {
  it("renders a created arc with its deployment state", async () => {
    const arc = await createArc();
    await renderToolbar();
    expect(await screen.findByText(arc.name)).toBeTruthy();
    expect((await screen.findAllByText("Stopped")).length).toBeGreaterThan(0);
  });

  it("places the explorer layout from the explorer action", async () => {
    const { store } = await renderToolbar();
    await waitFor(() => getIconButton(document.body, "explore"));
    fireEvent.click(getIconButton(document.body, "explore"));
    await waitForPlacedLayout(store, Arc.EXPLORER_LAYOUT_TYPE);
  });

  it("creates an arc through the create action's modal", async () => {
    const { store } = await renderToolbar();
    await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(getIconButton(document.body, "add"));
    const input = await screen.findByPlaceholderText("Automation Name");
    const name = uniqueName("arc");
    fireEvent.change(input, { target: { value: name } });
    const create = screen.getByRole("button", { name: "Create" });
    await waitFor(() => expect(create.className).not.toContain("pluto--disabled"));
    fireEvent.click(create);
    await waitFor(() => {
      const placed = Session.Layout.selectByFilter(
        store.getState(),
        (l) => l.type === Arc.EDITOR_LAYOUT_TYPE && l.name === name,
      );
      if (placed == null) throw new Error(`no arc layout named ${name}`);
    });
    await waitFor(async () => {
      const arcs = await client.arcs.retrieve({ names: [name] });
      expect(arcs.length).toBe(1);
    });
  });

  it("opens the arc editor on double click", async () => {
    const arc = await createArc();
    const { store } = await renderToolbar();
    fireEvent.doubleClick(await screen.findByText(arc.name));
    const key = await waitForPlacedLayout(store, Arc.EDITOR_LAYOUT_TYPE);
    expect(key).toBe(arc.key);
  });

  describe("context menu", () => {
    it("places the editor layout from Edit", async () => {
      const arc = await createArc();
      const { store } = await renderToolbar();
      fireEvent.contextMenu(await screen.findByText(arc.name));
      fireEvent.click(await screen.findByText("Edit"));
      const key = await waitForPlacedLayout(store, Arc.EDITOR_LAYOUT_TYPE);
      expect(key).toBe(arc.key);
    });

    it("renames the arc through the inline editor", async () => {
      const arc = await createArc();
      await renderToolbar();
      fireEvent.contextMenu(await screen.findByText(arc.name));
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditing(`text-${arc.key}`);
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(async () =>
        expect((await client.arcs.retrieve({ key: arc.key })).name).toBe(renamed),
      );
    });
  });
});
