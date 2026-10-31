// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, type Synnax as Client, view } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Component } from "@synnaxlabs/lyra/component";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Ranger } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { uniqueRangeName } from "@/platform/range/testutil";
import { View } from "@/platform/view";
import { enableEditing } from "@/platform/view/testutil";
import {
  awaitTextEditing,
  commitTextEdit,
  countEditableText,
  createConsoleWrapper,
  createTestClientWithGrants,
  getBySelector,
  getIconButton,
  type Grants,
  openContextMenu,
  queryIconButton,
  uniqueName,
} from "@/testutil";

const RenderItem = ({ itemKey }: List.ItemProps<ranger.Key>): ReactElement => (
  <span>{String(itemKey)}</span>
);
const item = Component.renderProp(RenderItem);

const Inner = (): ReactElement => {
  const listProps = Ranger.useList({
    initialQuery: { where: View.useContext().getInitialView().where },
  });
  return (
    <View.Form {...listProps}>
      <View.Toolbar>
        <View.FilterMenu>
          <Menu.Item itemKey="only">Filter Option</Menu.Item>
        </View.FilterMenu>
        <View.Search />
      </View.Toolbar>
      <View.Items<ranger.Key>>{item}</View.Items>
    </View.Form>
  );
};

const Harness = (): ReactElement => (
  <View.Frame resourceType="range" icon="Range">
    <Inner />
  </View.Frame>
);
Harness.displayName = "Harness";

const client = createTestClient();

/** @returns the name of the tab the view strip shows as selected. */
const selectedViewName = (): string | null =>
  screen.getAllByRole("tab").find((tab) => tab.getAttribute("aria-selected") === "true")
    ?.textContent ?? null;

const createView = async () =>
  await client.views.create({
    name: uniqueName("view"),
    type: "range",
  });

const renderHarness = async (as: Client = client): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  render(<Harness />, { wrapper });
};

describe("View", () => {
  it("hides the search input until the view is made editable", async () => {
    await renderHarness();
    await waitFor(() => expect(screen.getByText("All ranges")).toBeTruthy());
    expect(screen.queryByPlaceholderText("Search ranges...")).toBeNull();
  });

  it("renders the search input once editing is enabled", async () => {
    await renderHarness();
    await enableEditing();
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Search ranges...")).toBeTruthy(),
    );
  });

  it("filters the listed items down to those matching the search term", async () => {
    const start = TimeStamp.now();
    const timeRange = new TimeRange(start, start.add(TimeSpan.seconds(10)));
    const alpha = await client.ranges.create({
      name: uniqueRangeName("alpha"),
      timeRange,
    });
    const bravo = await client.ranges.create({
      name: uniqueRangeName("bravo"),
      timeRange,
    });
    await renderHarness();
    await enableEditing();
    const input = await waitFor(() =>
      screen.getByPlaceholderText<HTMLInputElement>("Search ranges..."),
    );
    fireEvent.change(input, { target: { value: alpha.name } });
    await waitFor(() => {
      expect(screen.getByText(String(alpha.key))).toBeTruthy();
      expect(screen.queryByText(String(bravo.key))).toBeNull();
    });
  });

  it("shows the empty state when the search matches no ranges", async () => {
    await renderHarness();
    await enableEditing();
    const input = await waitFor(() =>
      screen.getByPlaceholderText<HTMLInputElement>("Search ranges..."),
    );
    // Core search fuzzy-matches per token: dictionary-ish words ("no", "range") hit
    // real ranges, so the term must be pure gibberish.
    fireEvent.change(input, { target: { value: uniqueName("zzqjxvwq") } });
    await waitFor(() => expect(screen.getByText("No ranges found")).toBeTruthy());
  });

  it("lays the views out as a tab strip", async () => {
    const saved = await createView();
    await renderHarness();
    const tab = await screen.findByRole("tab", { name: saved.name });
    expect(screen.getByRole("tablist").contains(tab)).toBe(true);
    expect(selectedViewName()).toBe("All ranges");
  });

  it("moves the selection to a view whose tab is clicked", async () => {
    const saved = await createView();
    await renderHarness();
    const tab = await screen.findByRole("tab", { name: saved.name });
    fireEvent.click(tab);
    await waitFor(() => expect(selectedViewName()).toBe(saved.name));
  });

  it("leaves a static view's name plain while a saved view's is editable", async () => {
    const saved = await createView();
    await renderHarness();
    await screen.findByText(saved.name);
    await waitFor(() => expect(countEditableText(List.itemNameID(saved.key))).toBe(1));
    expect(screen.getByText("All ranges").className).not.toContain(
      "lyra-text--editable",
    );
  });

  it("renames a saved view in place from the context menu", async () => {
    const saved = await createView();
    await renderHarness();
    fireEvent.contextMenu(await screen.findByText(saved.name));
    fireEvent.click(await screen.findByText("Rename"));
    const editor = await awaitTextEditing(List.itemNameID(saved.key));
    const renamed = uniqueName("renamed");
    commitTextEdit(editor, renamed);
    await waitFor(async () =>
      expect((await client.views.retrieve(saved.key)).name).toBe(renamed),
    );
  });

  it("withholds rename from a static view's context menu", async () => {
    await createView();
    await renderHarness();
    fireEvent.contextMenu(await screen.findByText("All ranges"));
    expect(await screen.findByText("Reload Console")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("reveals the filter menu contents when the filter trigger is opened", async () => {
    await renderHarness();
    await enableEditing();
    const trigger = await waitFor(() => getIconButton(document.body, "filter"));
    fireEvent.click(trigger);
    await waitFor(() => expect(screen.getByText("Filter Option")).toBeTruthy());
  });
});

describe("View permissions", () => {
  const createSubject = async (grants: Grants) =>
    await createTestClientWithGrants(client, {
      ...grants,
      retrieve: [ranger.TYPE_ONTOLOGY_ID, view.TYPE_ONTOLOGY_ID],
    });

  it("should leave the view read-only for a subject who cannot update views", async () => {
    await renderHarness(await createSubject({}));
    await enableEditing();
    await screen.findByText("All ranges");
    expect(screen.queryByPlaceholderText("Search ranges...")).toBeNull();
    expect(
      queryIconButton(getBySelector(document.body, ".console-controls"), "add"),
    ).toBeNull();
  });

  it("should withhold the view create button from a subject who cannot create views", async () => {
    await renderHarness(await createSubject({ update: [view.TYPE_ONTOLOGY_ID] }));
    await enableEditing();
    await screen.findByPlaceholderText("Search ranges...");
    expect(
      queryIconButton(getBySelector(document.body, ".console-controls"), "add"),
    ).toBeNull();
  });

  it("should withhold rename from a subject who cannot update views", async () => {
    const saved = await createView();
    await renderHarness(await createSubject({ delete: [view.TYPE_ONTOLOGY_ID] }));
    fireEvent.contextMenu(await screen.findByText(saved.name));
    // Delete shares the selection the rename item needs, so its presence proves the
    // menu resolved before the absence below is read.
    expect(await screen.findByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(countEditableText(List.itemNameID(saved.key))).toBe(0);
  });

  it("should offer the view create button to a subject who may create views", async () => {
    await renderHarness(
      await createSubject({
        update: [view.TYPE_ONTOLOGY_ID],
        create: [view.TYPE_ONTOLOGY_ID],
      }),
    );
    await enableEditing();
    await waitFor(() =>
      expect(
        queryIconButton(getBySelector(document.body, ".console-controls"), "add"),
      ).toBeTruthy(),
    );
  });

  describe("context menu", () => {
    const createView = async (): Promise<string> => {
      const { name } = await client.views.create({
        name: uniqueName("view"),
        type: "range",
      });
      return name;
    };

    it("should withhold the delete item from a subject who cannot delete views", async () => {
      const name = await createView();
      await renderHarness(
        await createSubject({
          update: [view.TYPE_ONTOLOGY_ID],
          create: [view.TYPE_ONTOLOGY_ID],
        }),
      );
      await enableEditing();
      // The create button appearing proves the grant queries have answered, so the
      // missing delete item is the gate rather than a check still in flight.
      await waitFor(() =>
        expect(
          queryIconButton(getBySelector(document.body, ".console-controls"), "add"),
        ).toBeTruthy(),
      );
      await openContextMenu(name);
      expect(await screen.findByText("Rename")).toBeTruthy();
      expect(screen.queryByText("Delete")).toBeNull();
    });

    it("should offer the delete item to a subject who may delete views", async () => {
      const name = await createView();
      await renderHarness(await createSubject({ delete: [view.TYPE_ONTOLOGY_ID] }));
      await openContextMenu(name);
      expect(await screen.findByText("Delete")).toBeTruthy();
    });
  });
});
