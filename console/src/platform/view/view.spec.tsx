// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, ranger, type Synnax as Client, view } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Access, Flux } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it, onTestFinished } from "vitest";

import { Label } from "@/platform/label";
import { Modals } from "@/platform/modals";
import { uniqueRangeName } from "@/platform/range/testutil";
import { View } from "@/platform/view";
import { createTabWrapper, type TabWrapper } from "@/platform/view/testutil";
import { Session } from "@/session";
import {
  awaitTextEditing,
  awaitTextEditingElement,
  commitTextEdit,
  countEditableText,
  createTestClientWithGrants,
  getBySelector,
  getIconButton,
  type Grants,
  openContextMenu,
  queryIconButton,
  renderSuspended,
  uniqueName,
} from "@/testutil";

const RenderItem = ({ itemKey }: List.ItemProps<ranger.Key>): ReactElement => (
  <span>{String(itemKey)}</span>
);
const item = Component.renderProp(RenderItem);

const PROPERTIES: View.Filter.Properties<ranger.Where> = {
  labels: Label.Filter.createProperty(),
};

const Inner = (): ReactElement => {
  const [where, setWhere] = useState(View.useContext().getInitialView().where);
  const listProps = Flux.useList((client) => client.ranges, {
    name: "ranges",
    query: { where },
  });
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  return (
    <View.Form {...listProps} onWhereChange={setWhere} itemHeight={40}>
      <View.Toolbar>
        <View.Filter.Menu properties={PROPERTIES} />
        {hasCreatePermission && (
          <Button.Button tooltip="Create range">
            <Icon.Add />
          </Button.Button>
        )}
      </View.Toolbar>
      <View.Items<ranger.Key>>{item}</View.Items>
    </View.Form>
  );
};

const TAB_TYPE = "explorer";

const EXPLORER: panel.NewTab = { variant: "view", type: TAB_TYPE };

const Harness = (): ReactElement => (
  <View.Frame resourceType="range" icon="Range" tabType={TAB_TYPE}>
    <Inner />
  </View.Frame>
);
Harness.displayName = "Harness";

const client = createTestClient();

/** @returns the name of the tab the view strip shows as selected. */
const selectedViewName = (): string | null =>
  screen.getAllByRole("tab").find((tab) => tab.getAttribute("aria-selected") === "true")
    ?.textContent ?? null;

/**
 * Creates a range view that is deleted when the test ends. The strip loads a page of
 * views, so views left behind would push new ones off it.
 */
const createView = async (): Promise<view.View> => {
  const created = await client.views.create({
    name: uniqueName("view"),
    type: "range",
  });
  onTestFinished(async () => await client.views.delete(created.key));
  return created;
};

/** @returns a tab that shows the saved view. */
const showing = (saved: view.View): panel.NewTab => ({
  variant: "resource",
  resource: view.resourceID(saved.key),
});

const renderHarness = async (
  as: Client = client,
  tab: panel.NewTab = EXPLORER,
  beside?: panel.NewTab[],
): Promise<TabWrapper> => {
  const wrapped = await createTabWrapper({ client, as, tab, beside });
  await renderSuspended(
    <>
      <Harness />
      <Modals.Stack />
    </>,
    { wrapper: wrapped.wrapper },
  );
  return wrapped;
};

/**
 * Clicks the chip of the view with the name. Waits first for the create button, which
 * shows that the policies of a subject who may create views have loaded.
 */
const clickView = async (name: string): Promise<void> => {
  await waitFor(() => getIconButton(getViews(), "add"));
  fireEvent.click(await screen.findByRole("tab", { name }));
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
    await clickView(saved.name);
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

  it("renames the copy that was right-clicked when two strips show the view", async () => {
    const saved = await createView();
    const { wrapper } = await createTabWrapper({ client, tab: EXPLORER });
    await renderSuspended(
      <>
        <Harness />
        <Harness />
      </>,
      { wrapper },
    );
    const copies = await waitFor(() => {
      const found = screen.getAllByRole("tab", { name: saved.name });
      expect(found).toHaveLength(2);
      return found;
    });
    fireEvent.contextMenu(copies[1]);
    fireEvent.click(await screen.findByText("Rename"));
    const editor = await awaitTextEditingElement();
    expect(copies[1].contains(editor)).toBe(true);
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
    await clickView(saved.name);
    await waitFor(() => expect(selectedViewName()).toBe(saved.name));
    expect(
      screen.getByPlaceholderText<HTMLInputElement>("Search ranges...").value,
    ).toBe("");
  });

  it("reveals the filter menu contents when the filter trigger is opened", async () => {
    await renderHarness();
    const trigger = await waitFor(() => getIconButton(document.body, "filter"));
    fireEvent.click(trigger);
    await screen.findByRole("menuitem", { name: "Labels" });
  });
});

/** @returns the tab of the wrapper, as the Core stores it. */
const retrieveTab = async ({
  panelKey,
  tabKey,
}: TabWrapper): Promise<panel.Tab | undefined> =>
  panel.findTab((await client.panels.retrieve(panelKey)).root, tabKey);

describe("View open in a tab", () => {
  it("shows the saved view its tab holds", async () => {
    const saved = await createView();
    await renderHarness(client, showing(saved));
    await waitFor(() => expect(selectedViewName()).toBe(saved.name));
  });

  it("stores a view opened from the strip in its tab", async () => {
    const saved = await createView();
    const wrapped = await renderHarness();
    await clickView(saved.name);
    await waitFor(() => expect(selectedViewName()).toBe(saved.name));
    await waitFor(async () =>
      expect(await retrieveTab(wrapped)).toEqual({
        variant: "resource",
        key: wrapped.tabKey,
        resource: view.resourceID(saved.key),
      }),
    );
  });

  it("stores the static view in its tab when it opens again", async () => {
    const saved = await createView();
    const wrapped = await renderHarness(client, showing(saved));
    await clickView("All ranges");
    await waitFor(() => expect(selectedViewName()).toBe("All ranges"));
    await waitFor(async () =>
      expect(await retrieveTab(wrapped)).toEqual({
        variant: "view",
        key: wrapped.tabKey,
        type: TAB_TYPE,
      }),
    );
  });

  it("focuses the tab that already shows a view instead of showing it twice", async () => {
    const saved = await createView();
    const other = uuid.create();
    const wrapped = await renderHarness(client, EXPLORER, [
      { ...showing(saved), key: other },
    ]);
    await clickView(saved.name);
    await waitFor(() =>
      expect(
        Session.Panel.selectSelectedTabs(wrapped.store.getState(), wrapped.panelKey),
      ).toEqual([other]),
    );
    expect(selectedViewName()).toBe("All ranges");
    expect(await retrieveTab(wrapped)).toEqual({
      variant: "view",
      key: wrapped.tabKey,
      type: TAB_TYPE,
    });
  });

  it("keeps the view of a subject who may not change the panel", async () => {
    const saved = await createView();
    const subject = await createTestClientWithGrants(client, {
      retrieve: [
        ranger.TYPE_RESOURCE_ID,
        view.TYPE_RESOURCE_ID,
        panel.TYPE_RESOURCE_ID,
      ],
    });
    const wrapped = await renderHarness(subject);
    fireEvent.click(await screen.findByRole("tab", { name: saved.name }));
    await act(async () => {});
    expect(selectedViewName()).toBe("All ranges");
    expect(await retrieveTab(wrapped)).toEqual({
      variant: "view",
      key: wrapped.tabKey,
      type: TAB_TYPE,
    });
  });
});

/** @returns the tabs of the panel beside the tab of the wrapper. */
const tabsBeside = async ({ panelKey, tabKey }: TabWrapper): Promise<panel.Tab[]> => {
  const { root } = await client.panels.retrieve(panelKey);
  return (panel.findTabLeaf(root, tabKey)?.tabs ?? []).filter((t) => t.key !== tabKey);
};

/** Waits for one tab beside the tab of the wrapper, and returns it. */
const waitForNewTab = async (wrapped: TabWrapper): Promise<panel.Tab> =>
  await waitFor(async () => {
    const added = await tabsBeside(wrapped);
    expect(added).toHaveLength(1);
    return added[0];
  });

const roles = new RoleClients(client);

describe("View open in a new tab", () => {
  it("opens a view in a new tab from its context menu", async () => {
    const saved = await createView();
    const wrapped = await renderHarness();
    fireEvent.contextMenu(await screen.findByText(saved.name));
    fireEvent.click(await screen.findByText("Open in new tab"));
    expect(await waitForNewTab(wrapped)).toMatchObject(showing(saved));
    expect(selectedViewName()).toBe("All ranges");
  });

  it("opens a view in a new tab on a Cmd+click", async () => {
    const saved = await createView();
    const wrapped = await renderHarness();
    await waitFor(() => getIconButton(getViews(), "add"));
    fireEvent.click(await screen.findByRole("tab", { name: saved.name }), {
      metaKey: true,
    });
    expect(await waitForNewTab(wrapped)).toMatchObject(showing(saved));
    expect(selectedViewName()).toBe("All ranges");
  });

  it("opens the static view in a new tab", async () => {
    const wrapped = await renderHarness();
    fireEvent.contextMenu(await screen.findByText("All ranges"));
    fireEvent.click(await screen.findByText("Open in new tab"));
    expect(await waitForNewTab(wrapped)).toMatchObject({
      variant: "view",
      type: TAB_TYPE,
    });
  });

  it("offers no new tab for the saved view its tab shows", async () => {
    const saved = await createView();
    await renderHarness(client, showing(saved));
    await waitFor(() => expect(selectedViewName()).toBe(saved.name));
    fireEvent.contextMenu(screen.getByRole("tab", { name: saved.name }));
    // Both grants answer from one policy read, so this item shows that it landed.
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.queryByText("Open in new tab")).toBeNull();
  });

  it("offers no new tab to a user who may not change the panel", async () => {
    const saved = await createView();
    const wrapped = await renderHarness(await roles.get("Viewer"));
    fireEvent.contextMenu(await screen.findByText(saved.name));
    expect(await screen.findByText("Reload Console")).toBeTruthy();
    expect(screen.queryByText("Open in new tab")).toBeNull();
    fireEvent.click(screen.getByRole("tab", { name: saved.name }), { metaKey: true });
    await act(async () => {});
    expect(selectedViewName()).toBe("All ranges");
    expect(await tabsBeside(wrapped)).toEqual([]);
  });
});

describe("View permissions", () => {
  const createSubject = async (grants: Grants) =>
    await createTestClientWithGrants(client, {
      ...grants,
      retrieve: [
        ranger.TYPE_RESOURCE_ID,
        view.TYPE_RESOURCE_ID,
        panel.TYPE_RESOURCE_ID,
      ],
    });

  it("should let a subject who cannot update views search and filter a static view", async () => {
    const saved = await createView();
    await renderHarness(await createSubject({ delete: [view.TYPE_RESOURCE_ID] }));
    await openContextMenu(saved.name);
    // Both grants answer from one policy read, so this item shows that it landed.
    await screen.findByText("Delete");
    expect(selectedViewName()).toBe("All ranges");
    expect(screen.getByPlaceholderText("Search ranges...")).toBeTruthy();
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
    await renderHarness(await createSubject({}), showing(saved));
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

  it("should keep the toolbar of a saved view a subject cannot update, without its filters", async () => {
    const saved = await createView();
    await renderHarness(
      await createSubject({ create: [ranger.TYPE_RESOURCE_ID] }),
      showing(saved),
    );
    await waitFor(() => expect(selectedViewName()).toBe(saved.name));
    await waitFor(() =>
      expect(
        queryIconButton(getBySelector(document.body, ".console-view__toolbar"), "add"),
      ).toBeTruthy(),
    );
    expect(queryIconButton(document.body, "filter")).toBeNull();
  });

  it("should withhold the view create button from a subject who cannot create views", async () => {
    const saved = await createView();
    await renderHarness(
      await createSubject({ update: [view.TYPE_RESOURCE_ID] }),
      showing(saved),
    );
    await waitFor(() => expect(selectedViewName()).toBe(saved.name));
    // Both grants answer from one policy read, so this button shows that it landed.
    await waitFor(() => expect(queryIconButton(document.body, "filter")).toBeTruthy());
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
    const createViewName = async (): Promise<string> => (await createView()).name;

    it("should withhold the delete item from a subject who cannot delete views", async () => {
      const name = await createViewName();
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
      const name = await createViewName();
      await renderHarness(await createSubject({ delete: [view.TYPE_RESOURCE_ID] }));
      await openContextMenu(name);
      expect(await screen.findByText("Delete")).toBeTruthy();
    });
  });
});
