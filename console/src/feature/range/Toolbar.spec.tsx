// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  label,
  lineplot,
  NotFoundError,
  panel,
  project,
  ranger,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import { Session } from "@/session";
import {
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  createTestClientWithGrants,
  getIconButton,
  type Grants,
  openContextMenu,
  resolveFocusedTab,
  selectTestProject,
  type TestStore,
} from "@/testutil";

const client = createTestClient();

const expectFocusedRange = async (store: TestStore, key: string): Promise<void> => {
  const tab = await resolveFocusedTab(store, client);
  if (tab.variant !== "resource")
    throw new Error("focused tab is not a range resource");
  expect(tab.resource.type).toBe(ranger.TYPE_ONTOLOGY_ID.type);
  expect(tab.resource.key).toBe(key);
};

const createLocalRangeState = (name: string): Session.Range.StaticState => {
  const start = TimeStamp.now();
  return {
    key: uuid.create(),
    name,
    persisted: false,
    variant: "static",
    timeRange: { start: Number(start), end: Number(start.add(TimeSpan.seconds(1))) },
  };
};

const toState = (rng: ranger.Range): Session.Range.State =>
  Session.Range.fromClient(rng.payload)[0];

interface RenderToolbarOptions {
  ranges?: Session.Range.State[];
  active?: string;
  /** The client the toolbar renders against; defaults to the root client. */
  as?: Client;
}

const renderToolbar = async ({
  ranges = [],
  active,
  as = client,
}: RenderToolbarOptions = {}): Promise<{ store: TestStore }> => {
  const { wrapper, store } = await createConsoleWrapper({ client: as });
  await selectTestProject(store, client);
  render(
    <>
      {Range.TOOLBAR.content}
      <Modals.Stack />
    </>,
    { wrapper },
  );
  if (ranges.length > 0) store.dispatch(Session.Range.add(ranges));
  if (active != null) store.dispatch(Session.Range.select(active));
  else store.dispatch(Session.Range.clearSelected());
  return { store };
};

describe("range/Toolbar", () => {
  it("shows the empty state and opens the Range Explorer from it", async () => {
    const { store } = await renderToolbar();
    expect(await screen.findByText("No favorited ranges")).toBeTruthy();
    fireEvent.click(await screen.findByText("Open range explorer"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "view")
      throw new Error("focused tab is not the range explorer view");
    expect(tab.type).toBe(Range.Explorer.TAB_TYPE);
  });

  it("renders favorited ranges and marks local ones with the L badge", async () => {
    const persisted = toState(await createTestRange(client));
    const local = createLocalRangeState(uniqueRangeName("local"));
    await renderToolbar({ ranges: [persisted, local] });
    expect(await screen.findByText(persisted.name)).toBeTruthy();
    expect(await screen.findByText(local.name)).toBeTruthy();
    expect(screen.getByText("L")).toBeTruthy();
  });

  it("fades a range name too long for the toolbar", async () => {
    const rng = createLocalRangeState(uniqueRangeName("a-very-long-range-name"));
    await renderToolbar({ ranges: [rng] });
    const name = await screen.findByText(rng.name);
    expect(name.className).toContain("pluto-text--overflow-fade");
    expect(name.closest(".console-range-list-item__name")).not.toBeNull();
  });

  it("sets the clicked range as active", async () => {
    const rng = toState(await createTestRange(client));
    const { store } = await renderToolbar({ ranges: [rng] });
    await screen.findByText(rng.name);
    // Re-query synchronously: async permission resolution can replace the text node,
    // detaching a match held across an await before the click lands.
    fireEvent.click(screen.getByText(rng.name));
    await waitFor(() =>
      expect(Session.Range.selectSelectedKey(store.getState())).toBe(rng.key),
    );
  });

  describe("context menu", () => {
    it("sets a non-active range as active", async () => {
      const rng = toState(await createTestRange(client));
      const { store } = await renderToolbar({ ranges: [rng] });
      await openContextMenu(rng.name);
      fireEvent.click(await screen.findByText("Set as active range"));
      await waitFor(() =>
        expect(Session.Range.selectSelectedKey(store.getState())).toBe(rng.key),
      );
    });

    it("clears the active range", async () => {
      const rng = toState(await createTestRange(client));
      const { store } = await renderToolbar({ ranges: [rng], active: rng.key });
      await openContextMenu(rng.name);
      fireEvent.click(await screen.findByText("Clear active range"));
      await waitFor(() =>
        expect(Session.Range.selectSelectedKey(store.getState())).toBeUndefined(),
      );
    });

    it("opens the overview tab from View details", async () => {
      const rng = toState(await createTestRange(client));
      const { store } = await renderToolbar({ ranges: [rng] });
      await openContextMenu(rng.name);
      fireEvent.click(await screen.findByText("View details"));
      await expectFocusedRange(store, rng.key);
    });

    it("removes the range from favorites without deleting it on the server", async () => {
      const created = await createTestRange(client);
      const rng = toState(created);
      const { store } = await renderToolbar({ ranges: [rng] });
      await openContextMenu(rng.name);
      fireEvent.click(await screen.findByText("Unfavorite"));
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), rng.key)).toBeUndefined(),
      );
      expect((await client.ranges.retrieve(created.key)).name).toBe(rng.name);
    });

    it("persists a local range to the cluster via Save to Core", async () => {
      const local = createLocalRangeState(uniqueRangeName("save"));
      const { store } = await renderToolbar({ ranges: [local] });
      await openContextMenu(local.name);
      fireEvent.click(await screen.findByText("Save to Core"));
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), local.key)?.persisted).toBe(
          true,
        ),
      );
      await waitFor(async () =>
        expect((await client.ranges.retrieve(local.key)).name).toBe(local.name),
      );
    });

    it("deletes a persisted range after confirmation", async () => {
      const created = await createTestRange(client);
      const rng = toState(created);
      const { store } = await renderToolbar({ ranges: [rng] });
      await openContextMenu(rng.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${rng.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), rng.key)).toBeUndefined(),
      );
      await waitFor(async () => {
        await expect(client.ranges.retrieve(created.key)).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
    });

    it("renames a persisted range through the inline editor", async () => {
      const created = await createTestRange(client);
      const rng = toState(created);
      const { store } = await renderToolbar({ ranges: [rng] });
      await openContextMenu(rng.name);
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditing(`text-${rng.key}`);
      const renamed = uniqueRangeName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), rng.key)?.name).toBe(
          renamed,
        ),
      );
      await waitFor(async () =>
        expect((await client.ranges.retrieve(created.key)).name).toBe(renamed),
      );
    });

    it("renames a local range without reaching the Core", async () => {
      const local = createLocalRangeState(uniqueRangeName("local"));
      const { store } = await renderToolbar({ ranges: [local] });
      await openContextMenu(local.name);
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditing(`text-${local.key}`);
      const renamed = uniqueRangeName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), local.key)?.name).toBe(
          renamed,
        ),
      );
      await expect(client.ranges.retrieve(local.key)).rejects.toThrow(NotFoundError);
    });

    it("withholds the active-plot item when the focused tab is not a plot", async () => {
      const { store } = await renderToolbar();
      fireEvent.click(await screen.findByText("Open range explorer"));
      await resolveFocusedTab(store, client);
      const rng = await createTestRange(client);
      store.dispatch(Session.Range.add([toState(rng)]));
      await openContextMenu(rng.name);
      expect(await screen.findByText("Add to new plot")).toBeTruthy();
      expect(screen.queryByText("Add to active plot")).toBeNull();
    });

    it("opens the create modal for a child range", async () => {
      const rng = toState(await createTestRange(client));
      await renderToolbar({ ranges: [rng] });
      await openContextMenu(rng.name);
      fireEvent.click(await screen.findByText("Create child range"));
      expect(await screen.findByText("Save locally")).toBeTruthy();
    });
  });

  it("opens the create range modal from the toolbar action", async () => {
    await renderToolbar();
    const action = await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(action);
    expect(await screen.findByText("Save locally")).toBeTruthy();
  });
});

describe("Range.fetchIfNotInState", () => {
  it("returns the state already in the store without hitting the cluster", async () => {
    const local = createLocalRangeState(uniqueRangeName("cached"));
    const { store } = await createConsoleWrapper({ client });
    store.dispatch(Session.Range.add(local));
    const result = await Range.fetchIfNotInState(store, client, local.key);
    expect(result.name).toBe(local.name);
  });

  it("fetches a missing range from the cluster and adds it to the store", async () => {
    const created = await createTestRange(client);
    const { store } = await createConsoleWrapper({ client });
    const result = await Range.fetchIfNotInState(store, client, created.key);
    expect(result.name).toBe(created.name);
    expect(Session.Range.selectState(store.getState(), created.key)?.persisted).toBe(
      true,
    );
  });
});

describe("range/ContextMenu permissions", () => {
  const createSubject = async (grants: Grants) =>
    await createTestClientWithGrants(client, {
      ...grants,
      retrieve: [
        ranger.TYPE_ONTOLOGY_ID,
        lineplot.TYPE_ONTOLOGY_ID,
        label.TYPE_ONTOLOGY_ID,
        project.TYPE_ONTOLOGY_ID,
        panel.TYPE_ONTOLOGY_ID,
      ],
    });

  it("should withhold every write from a subject with no grants", async () => {
    const rng = await createTestRange(client);
    await renderToolbar({ ranges: [toState(rng)], as: await createSubject({}) });
    await openContextMenu(rng.name);
    expect(await screen.findByText("Set as active range")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Create child range")).toBeNull();
    expect(screen.queryByText("Add to new plot")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should withhold the plot items from a subject who may only write ranges", async () => {
    const rng = await createTestRange(client);
    await renderToolbar({
      ranges: [toState(rng)],
      as: await createSubject({
        create: [ranger.TYPE_ONTOLOGY_ID],
        update: [ranger.TYPE_ONTOLOGY_ID],
      }),
    });
    await openContextMenu(rng.name);
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(await screen.findByText("Create child range")).toBeTruthy();
    expect(screen.queryByText("Add to new plot")).toBeNull();
  });

  it("should offer the plot item to a subject who may create plots", async () => {
    const rng = await createTestRange(client);
    await renderToolbar({
      ranges: [toState(rng)],
      as: await createSubject({ create: [lineplot.TYPE_ONTOLOGY_ID] }),
    });
    await openContextMenu(rng.name);
    expect(await screen.findByText("Add to new plot")).toBeTruthy();
  });
});
