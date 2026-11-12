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
import { Flux } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import { uniqueRangeName } from "@/platform/range/testutil";
import { View } from "@/platform/view";
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
  const [where, setWhere] = useState(View.useContext().getInitialView().where);
  const listProps = Flux.useList((client) => client.ranges, {
    name: "ranges",
    query: { where },
  });
  return (
    <View.Form {...listProps} onWhereChange={setWhere} itemHeight={40}>
      <View.Toolbar>
        <View.FilterMenu>
          <Menu.Item itemKey="only">Filter Option</Menu.Item>
        </View.FilterMenu>
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
  render(
    <>
      <Harness />
      <Modals.Stack />
    </>,
    { wrapper },
  );
};

/** Opens the create view modal. @returns its name input. */
const openCreateModal = async (): Promise<HTMLInputElement> => {
  const add = await waitFor(() => getIconButton(getViews(), "add"));
  fireEvent.click(add);
  return await screen.findByPlaceholderText<HTMLInputElement>("View name");
};

/** @returns the views bar of the frame. */
const getViews = (): HTMLElement =>
  getBySelector(document.body, ".console-view__views");

/** @returns the Create button of the create view modal. */
const getCreateButton = (): HTMLElement => {
  // The create view button names itself with a tooltip, and holds no text.
  const create = screen
    .getAllByRole("button", { name: /^Create/ })
    .find((b) => b.textContent?.startsWith("Create"));
  if (create == null) throw new Error("no Create button");
  return create;
};

/** @returns the icon the tab of the named view draws. */
const tabIcon = (name: string): SVGElement => {
  const tab = screen.getAllByRole("tab").find((t) => t.textContent === name);
  const svg = tab?.querySelector("svg");
  if (svg == null) throw new Error(`tab for ${name} draws no icon`);
  return svg;
};

describe("View", () => {
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
    const input =
      await screen.findByPlaceholderText<HTMLInputElement>("Search ranges...");
    fireEvent.change(input, { target: { value: alpha.name } });
    await waitFor(() => {
      expect(screen.getByText(String(alpha.key))).toBeTruthy();
      expect(screen.queryByText(String(bravo.key))).toBeNull();
    });
  });

  it("shows the empty state when the search matches no ranges", async () => {
    await renderHarness();
    const input =
      await screen.findByPlaceholderText<HTMLInputElement>("Search ranges...");
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

  it("creates a view with the name and icon picked in the create modal", async () => {
    await renderHarness();
    const input = await openCreateModal();
    expect(input.value).toBe("");
    fireEvent.click(screen.getByLabelText("Change icon"));
    const valve = await screen.findByLabelText("Valve");
    await act(async () => fireEvent.click(valve));
    const name = uniqueName("created");
    fireEvent.change(input, { target: { value: name } });
    fireEvent.click(getCreateButton());
    await waitFor(() => expect(selectedViewName()).toBe(name));
    expect(tabIcon(name).classList).toContain("lyra-icon--valve");
    const stored = await client.views.retrieve({ where: { name: { eq: name } } });
    expect(stored.map(({ icon }) => icon)).toEqual(["valve"]);
  });

  it("shows an empty name error at the field and keeps create enabled", async () => {
    await renderHarness();
    await openCreateModal();
    const create = getCreateButton();
    expect(create.getAttribute("aria-disabled")).toBeNull();
    fireEvent.click(create);
    expect(await screen.findByText("Name is required")).toBeTruthy();
    expect(screen.getByPlaceholderText("View name")).toBeTruthy();
  });

  it("draws the explorer icon on a static tab and a view icon on a saved one", async () => {
    const saved = await createView();
    await renderHarness();
    await screen.findByText(saved.name);
    expect(tabIcon("All ranges").classList).toContain("lyra-icon--range");
    expect(tabIcon(saved.name).classList).toContain("lyra-icon--view");
  });

  it("sets a saved view's icon from the change icon item", async () => {
    const saved = await createView();
    await renderHarness();
    fireEvent.contextMenu(await screen.findByText(saved.name));
    fireEvent.click(await screen.findByText("Change icon"));
    await screen.findByRole("dialog", { name: "View icon" });
    await act(async () => {
      fireEvent.click(screen.getByLabelText("Valve"));
    });
    await waitFor(() =>
      expect(tabIcon(saved.name).classList).toContain("lyra-icon--valve"),
    );
    expect((await client.views.retrieve(saved.key)).icon).toBe("valve");
  });

  it("withholds change icon from a static view's context menu", async () => {
    await renderHarness();
    fireEvent.contextMenu(await screen.findByText("All ranges"));
    expect(await screen.findByText("Reload Console")).toBeTruthy();
    expect(screen.queryByText("Change icon")).toBeNull();
  });

  it("leads the views with the search and ends them with the create button", async () => {
    await renderHarness();
    const input = await screen.findByPlaceholderText("Search ranges...");
    expect(input.closest(".console-view__search")).not.toBeNull();
    await waitFor(() => expect(queryIconButton(getViews(), "add")).toBeTruthy());
  });

  it("clears the search term of the views bar when another view opens", async () => {
    const saved = await createView();
    await renderHarness();
    const input =
      await screen.findByPlaceholderText<HTMLInputElement>("Search ranges...");
    fireEvent.change(input, { target: { value: "alpha" } });
    expect(input.value).toBe("alpha");
    fireEvent.click(await screen.findByRole("tab", { name: saved.name }));
    await waitFor(() => expect(selectedViewName()).toBe(saved.name));
    expect(
      screen.getByPlaceholderText<HTMLInputElement>("Search ranges...").value,
    ).toBe("");
  });

  it("reveals the filter menu contents when the filter trigger is opened", async () => {
    await renderHarness();
    const trigger = await waitFor(() => getIconButton(document.body, "filter"));
    fireEvent.click(trigger);
    await waitFor(() => expect(screen.getByText("Filter Option")).toBeTruthy());
  });
});

describe("View permissions", () => {
  const createSubject = async (grants: Grants) =>
    await createTestClientWithGrants(client, {
      ...grants,
      retrieve: [ranger.TYPE_RESOURCE_ID, view.TYPE_RESOURCE_ID],
    });

  it("should let a subject who cannot update views search and filter a static view", async () => {
    await renderHarness(await createSubject({}));
    await screen.findByPlaceholderText("Search ranges...");
    expect(queryIconButton(document.body, "filter")).toBeTruthy();
    expect(queryIconButton(getViews(), "add")).toBeNull();
  });

  it("should let a subject who cannot update views search a saved view but not filter it", async () => {
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
    const saved = await createView();
    await renderHarness(await createSubject({}));
    fireEvent.click(await screen.findByRole("tab", { name: saved.name }));
    await waitFor(() => expect(selectedViewName()).toBe(saved.name));
    fireEvent.change(screen.getByPlaceholderText("Search ranges..."), {
      target: { value: alpha.name },
    });
    await waitFor(() => {
      expect(screen.getByText(String(alpha.key))).toBeTruthy();
      expect(screen.queryByText(String(bravo.key))).toBeNull();
    });
    expect(queryIconButton(document.body, "filter")).toBeNull();
  });

  it("should withhold the view create button from a subject who cannot create views", async () => {
    await renderHarness(await createSubject({ update: [view.TYPE_RESOURCE_ID] }));
    await screen.findByPlaceholderText("Search ranges...");
    expect(queryIconButton(getViews(), "add")).toBeNull();
  });

  it("should withhold rename from a subject who cannot update views", async () => {
    const saved = await createView();
    await renderHarness(await createSubject({ delete: [view.TYPE_RESOURCE_ID] }));
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
        update: [view.TYPE_RESOURCE_ID],
        create: [view.TYPE_RESOURCE_ID],
      }),
    );
    await waitFor(() => expect(queryIconButton(getViews(), "add")).toBeTruthy());
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
          update: [view.TYPE_RESOURCE_ID],
          create: [view.TYPE_RESOURCE_ID],
        }),
      );
      // The create button appearing proves the grant queries have answered, so the
      // missing delete item is the gate rather than a check still in flight.
      await waitFor(() => expect(queryIconButton(getViews(), "add")).toBeTruthy());
      await openContextMenu(name);
      expect(await screen.findByText("Rename")).toBeTruthy();
      expect(screen.queryByText("Delete")).toBeNull();
    });

    it("should offer the delete item to a subject who may delete views", async () => {
      const name = await createView();
      await renderHarness(await createSubject({ delete: [view.TYPE_RESOURCE_ID] }));
      await openContextMenu(name);
      expect(await screen.findByText("Delete")).toBeTruthy();
    });
  });
});
