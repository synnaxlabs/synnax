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
  NotFoundError,
  ranger,
  type Synnax as Client,
  view,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List } from "@synnaxlabs/lyra/list";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Explorer } from "@/feature/range/explorer/Explorer";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { uniqueRangeName } from "@/platform/range/testutil";
import { Session } from "@/session";
import {
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  createTestClientWithGrants,
  renderSuspended,
  resolveFocusedTab,
  selectRange,
  selectTestProject,
  type TestStore,
  uniqueName,
  uniqueToken,
} from "@/testutil";

const client = createTestClient();

const renderExplorer = async (as: Client = client): Promise<TestStore> => {
  const { wrapper, store } = await createConsoleWrapper({ client: as });
  await renderSuspended(
    <>
      <Explorer />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  await screen.findByText("All ranges");
  return store;
};

const createRange = async (
  name: string,
  extra: Partial<ranger.New> = {},
): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name,
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(10))),
    ...extra,
  });
};

/**
 * Creates a label, one range with it, and one range without it. Both start years ahead
 * of the ranges other specs create, so they lead the newest-first list.
 */
const createLabeled = async (): Promise<{
  label: label.Label;
  labeled: ranger.Range;
  other: ranger.Range;
}> => {
  const label = await client.labels.create({
    name: uniqueName("label"),
    color: "#000000",
  });
  const start = TimeStamp.now().add(TimeSpan.days(3650));
  const timeRange = new TimeRange(start, start.add(TimeSpan.seconds(10)));
  const labeled = await createRange(uniqueRangeName("labeled"), {
    labels: [label.key],
    timeRange,
  });
  const other = await createRange(uniqueRangeName("other"), { timeRange });
  return { label, labeled, other };
};

const search = (term: string): void => {
  const input = screen.getByPlaceholderText<HTMLInputElement>("Search ranges...");
  fireEvent.change(input, { target: { value: term } });
};

/** @returns the list item of the named range. */
const itemOf = (name: string): HTMLElement => screen.getByRole("listitem", { name });

/** Searches for the range, then opens the context menu on it. */
const openMenuOf = async (rng: ranger.Range): Promise<void> => {
  search(rng.name);
  fireEvent.contextMenu(await screen.findByRole("listitem", { name: rng.name }));
};

const menuItem = async (name: string): Promise<HTMLElement> =>
  await screen.findByRole("menuitem", { name });

const SWAP = "console-range-filter__picker--swap";

/** jsdom lays nothing out, so a spec places the values beside the menu by hand. */
const BESIDE: DOMRect = {
  left: 150,
  top: 0,
  width: 200,
  height: 200,
  right: 350,
  bottom: 200,
  x: 150,
  y: 0,
  toJSON: () => "",
};

/** Opens the filter menu, then holds the clock so its timers run on demand. */
const openFilterMenu = async (): Promise<void> => {
  await renderExplorer();
  fireEvent.click(await screen.findByRole("button", { name: "Filter" }));
  await screen.findByRole("button", { name: "Labels", expanded: false });
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "performance"] });
};

/** @returns the closed property of the filter menu with the name. */
const closed = (name: string): HTMLElement =>
  screen.getByRole("button", { name, expanded: false });

/** @returns the open values of the property with the name. */
const values = (name: string): HTMLElement => screen.getByRole("dialog", { name });

const move = (target: HTMLElement, clientX: number, clientY: number): void => {
  fireEvent.pointerMove(target, { pointerType: "mouse", clientX, clientY });
};

const advance = (ms: number): void => {
  act(() => {
    vi.advanceTimersByTime(ms);
  });
};

/** Opens the labels, then moves the pointer across stage toward their values. */
const aimAtLabels = async (): Promise<void> => {
  await openFilterMenu();
  fireEvent.keyDown(closed("Labels"), { key: "ArrowRight" });
  vi.spyOn(values("Labels"), "getBoundingClientRect").mockReturnValue(BESIDE);
  move(screen.getByRole("button", { name: "Labels", expanded: true }), 100, 14);
  move(closed("Stage"), 140, 32);
};

describe("range/explorer/Explorer", () => {
  describe("filter", () => {
    it("should keep the ranges with a picked label and restore the rest when its chip goes", async () => {
      const { label, labeled, other } = await createLabeled();
      await renderExplorer();
      await screen.findByText(other.name);
      fireEvent.click(await screen.findByRole("button", { name: "Filter" }));
      const labels = await screen.findByRole("button", {
        name: "Labels",
        expanded: false,
      });
      fireEvent.keyDown(labels, { key: "ArrowRight" });
      fireEvent.change(await screen.findByPlaceholderText("Search labels..."), {
        target: { value: label.name },
      });
      fireEvent.click(await screen.findByRole("option", { name: label.name }));
      await waitFor(() => {
        expect(screen.queryByText(other.name)).toBeNull();
        expect(screen.getByText(labeled.name)).toBeTruthy();
      });
      fireEvent.click(screen.getByRole("button", { name: "Remove Labels filter" }));
      await screen.findByText(other.name);
    });

    it("should show the chip of a stored view that filters by one label", async () => {
      const { label, labeled, other } = await createLabeled();
      // A 0.61 view stores a sole labels filter outside an and.
      const stored = await client.views.create({
        name: uniqueName("view"),
        type: "range",
        where: { labels: [label.key] },
      });
      await renderExplorer();
      fireEvent.click(await screen.findByRole("tab", { name: stored.name }));
      await screen.findByText(labeled.name);
      expect(screen.getByRole("button", { name: "Remove Labels filter" })).toBeTruthy();
      expect(screen.queryByText(other.name)).toBeNull();
    });
  });

  describe("filter menu", () => {
    afterEach(() => {
      act(() => {
        vi.runOnlyPendingTimers();
      });
      vi.useRealTimers();
    });

    it("should open the values of a property once the pointer rests on it", async () => {
      await openFilterMenu();
      move(closed("Time"), 50, 70);
      advance(100);
      expect(values("Time").className).not.toContain(SWAP);
    });

    it("should not open the values of a property the pointer only passes", async () => {
      await openFilterMenu();
      move(closed("Stage"), 50, 42);
      advance(60);
      move(closed("Time"), 50, 70);
      advance(100);
      // Values that opened over the values of stage would carry the swap.
      expect(values("Time").className).not.toContain(SWAP);
      expect(screen.queryByRole("dialog", { name: "Stage" })).toBeNull();
    });

    it("should open the values on ArrowRight and close them on ArrowLeft", async () => {
      await openFilterMenu();
      const labels = closed("Labels");
      fireEvent.keyDown(labels, { key: "ArrowRight" });
      fireEvent.keyDown(screen.getByPlaceholderText("Search labels..."), {
        key: "ArrowLeft",
      });
      expect(screen.queryByRole("dialog", { name: "Labels" })).toBeNull();
      expect(document.activeElement).toBe(labels);
    });

    it("should keep the open values while the pointer crosses another property toward them", async () => {
      await aimAtLabels();
      advance(250);
      move(screen.getByPlaceholderText("Search labels..."), 200, 50);
      advance(300);
      expect(values("Labels")).toBeTruthy();
      expect(screen.queryByRole("dialog", { name: "Stage" })).toBeNull();
    });

    it("should swap in the values under the pointer once the aim lapses", async () => {
      await aimAtLabels();
      advance(300);
      expect(values("Stage").className).toContain(SWAP);
      expect(screen.queryByRole("dialog", { name: "Labels" })).toBeNull();
    });

    it("should end the aim when the pointer heads past the values", async () => {
      await aimAtLabels();
      move(closed("Stage"), 145, 180);
      advance(100);
      expect(values("Stage").className).toContain(SWAP);
    });

    it("should end the aim when the pointer drops without moving toward the values", async () => {
      await aimAtLabels();
      move(closed("Stage"), 141, 50);
      advance(100);
      expect(values("Stage").className).toContain(SWAP);
    });
  });

  describe("search", () => {
    it("should show a match under its parent and leave out the other children", async () => {
      const token = uniqueToken();
      const parent = await createRange(uniqueRangeName("parent"));
      const match = await createRange(`${token} match`, { parent: parent.key });
      const sibling = await createRange(uniqueRangeName("sibling"), {
        parent: parent.key,
      });
      await renderExplorer();
      search(token);
      await screen.findByText(match.name);
      await waitFor(() =>
        within(itemOf(parent.name)).getByRole("button", { name: "Collapse" }),
      );
      expect(screen.queryByText(sibling.name)).toBeNull();
    });

    it("should show the first 100 matches and say so", async () => {
      const token = uniqueToken();
      const start = TimeStamp.now();
      await client.ranges.create(
        Array.from({ length: 101 }, (_, i) => ({
          name: `${token} ${i}`,
          timeRange: new TimeRange(start, start.add(TimeSpan.seconds(10))),
        })),
      );
      await renderExplorer();
      search(token);
      await screen.findByText(/Showing the first 100 matches\./);
    });
  });

  describe("context menu", () => {
    it("should open the overview of the range", async () => {
      const rng = await createRange(uniqueRangeName("open"));
      const store = await renderExplorer();
      await selectTestProject(store, client);
      await openMenuOf(rng);
      fireEvent.click(await menuItem("Open"));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("focused tab is not a resource");
      expect(tab.resource).toEqual(ranger.resourceID(rng.key));
    });

    it("should rename the range in place", async () => {
      const rng = await createRange(uniqueRangeName("rename"));
      await renderExplorer();
      await openMenuOf(rng);
      fireEvent.click(await menuItem("Rename"));
      const renamed = uniqueRangeName("renamed");
      commitTextEdit(await awaitTextEditing(List.itemNameID(rng.key)), renamed);
      await waitFor(async () =>
        expect((await client.ranges.retrieve(rng.key)).name).toBe(renamed),
      );
    });

    it("should pin the range", async () => {
      const rng = await createRange(uniqueRangeName("pin"));
      const store = await renderExplorer();
      await openMenuOf(rng);
      fireEvent.click(await menuItem("Pin"));
      await waitFor(() =>
        expect(selectRange(store.getState(), rng.key)?.variant).toBe("persisted"),
      );
    });

    it("should open the create modal for a child range", async () => {
      const rng = await createRange(uniqueRangeName("parent"));
      await renderExplorer();
      await openMenuOf(rng);
      fireEvent.click(await menuItem("Start child range"));
      expect(await screen.findByText("Save locally")).toBeTruthy();
    });

    it("should delete the range after the confirm", async () => {
      const rng = await createRange(uniqueRangeName("delete"));
      const store = await renderExplorer();
      store.dispatch(Session.Range.add({ variant: "persisted", key: rng.key }));
      await openMenuOf(rng);
      fireEvent.click(await menuItem("Delete"));
      await screen.findByText(`Are you sure you want to delete ${rng.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(() =>
        expect(selectRange(store.getState(), rng.key)).toBeUndefined(),
      );
      await waitFor(
        async () =>
          await expect(client.ranges.retrieve(rng.key)).rejects.toThrow(NotFoundError),
      );
    });
  });

  describe("stored view", () => {
    it("should save the sort and the layout to the view", async () => {
      const stored = await client.views.create({
        name: uniqueName("view"),
        type: "range",
      });
      await renderExplorer();
      fireEvent.click(await screen.findByRole("tab", { name: stored.name }));
      fireEvent.click(await screen.findByRole("button", { name: /Newest first/ }));
      fireEvent.click(await screen.findByText("Name"));
      fireEvent.click(screen.getByRole("button", { name: /List/ }));
      await waitFor(async () => {
        const saved = await client.views.retrieve({ key: stored.key });
        if (saved.type !== "range") throw new Error("not a range view");
        expect([saved.sort, saved.layout]).toEqual(["name", "list"]);
      });
    });
  });

  describe("permissions", () => {
    /** @returns a client whose subject may read views but not update them. */
    const createReader = async (): Promise<Client> =>
      await createTestClientWithGrants(client, {
        retrieve: [
          ranger.TYPE_RESOURCE_ID,
          view.TYPE_RESOURCE_ID,
          label.TYPE_RESOURCE_ID,
        ],
        create: [ranger.TYPE_RESOURCE_ID],
      });

    it("should withhold the range writes from a subject with no range grants", async () => {
      const rng = await createRange(uniqueRangeName("read"));
      await renderExplorer(
        await createTestClientWithGrants(client, {
          retrieve: [ranger.TYPE_RESOURCE_ID, view.TYPE_RESOURCE_ID],
        }),
      );
      await openMenuOf(rng);
      await menuItem("Open");
      expect(screen.queryByRole("menuitem", { name: "Rename" })).toBeNull();
      expect(screen.queryByRole("menuitem", { name: "Start child range" })).toBeNull();
      expect(screen.queryByRole("menuitem", { name: "Delete" })).toBeNull();
    });

    it("should offer the range writes to a subject granted them", async () => {
      const rng = await createRange(uniqueRangeName("write"));
      await renderExplorer(
        await createTestClientWithGrants(client, {
          retrieve: [ranger.TYPE_RESOURCE_ID, view.TYPE_RESOURCE_ID],
          create: [ranger.TYPE_RESOURCE_ID],
          update: [ranger.TYPE_RESOURCE_ID],
          delete: [ranger.TYPE_RESOURCE_ID],
        }),
      );
      await openMenuOf(rng);
      expect(await menuItem("Rename")).toBeTruthy();
      expect(await menuItem("Start child range")).toBeTruthy();
      expect(await menuItem("Delete")).toBeTruthy();
    });

    it("should let a subject who cannot update views sort and lay out all ranges", async () => {
      await renderExplorer(await createReader());
      fireEvent.click(await screen.findByRole("button", { name: /Newest first/ }));
      fireEvent.click(await screen.findByText("Name"));
      fireEvent.click(screen.getByRole("button", { name: /List/ }));
      await waitFor(() => expect(screen.queryByText("Newest first")).toBeNull());
      expect(screen.getByRole("button", { name: /List/ }).className).toContain(
        "lyra--selected",
      );
      expect(screen.getByRole("button", { name: "Filter" })).toBeTruthy();
      expect(screen.getByPlaceholderText("Search ranges...")).toBeTruthy();
    });

    it("should show a stored view read-only to a subject who cannot update views", async () => {
      const { label: lbl, labeled } = await createLabeled();
      const stored = await client.views.create({
        name: uniqueName("view"),
        type: "range",
        where: { labels: [lbl.key] },
      });
      await renderExplorer(await createReader());
      fireEvent.click(await screen.findByRole("tab", { name: stored.name }));
      await screen.findByText(labeled.name);
      await screen.findByRole("button", { name: "Create range" });
      await act(async () => {});
      expect(screen.getByText("Labels")).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Remove Labels filter" })).toBeNull();
      expect(screen.queryByRole("button", { name: "Filter" })).toBeNull();
      expect(screen.getByPlaceholderText("Search ranges...")).toBeTruthy();
      fireEvent.click(screen.getByText("Newest first"));
      expect(screen.queryByText("Oldest first")).toBeNull();
      expect(screen.getByText("Tree")).toBeTruthy();
      expect(screen.queryByText("List")).toBeNull();
    });
  });
});
