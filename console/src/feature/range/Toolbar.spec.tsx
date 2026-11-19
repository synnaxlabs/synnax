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
  panel,
  project,
  ranger,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List } from "@synnaxlabs/lyra/list";
import { sleep, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, onTestFinished } from "vitest";

import { renderLinePlot } from "@/feature/lineplot/testutil";
import { Toolbar as LinePlotToolbar } from "@/feature/lineplot/toolbar";
import { Range } from "@/feature/range";
import { type Analytics } from "@/platform/analytics";
import { createTestSink, wrapWithSink } from "@/platform/analytics/testutil";
import { Modals } from "@/platform/modals";
import { uniqueRangeName } from "@/platform/range/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  createTestClientWithGrants,
  type Grants,
  HaulProvider,
  openContextMenu,
  resolveFocusedTab,
  selectTestProject,
  type TestStore,
} from "@/testutil";

const client = createTestClient();
// A separate client reads what the Core saved, not the rendering client's cache.
const reader = createTestClient();

const renderToolbar = async ({
  favorites = [],
  as = client,
  analytics,
}: { favorites?: ranger.Range[]; as?: Client; analytics?: Analytics.Sink } = {}) => {
  const { wrapper: base, store } = await createConsoleWrapper({ client: as });
  const wrapper = analytics == null ? base : wrapWithSink(base, analytics);
  await selectTestProject(store, client);
  favorites.forEach(({ key }) =>
    store.dispatch(Session.Range.add({ variant: "persisted", key })),
  );
  const content = (
    <HaulProvider>
      {Range.TOOLBAR.content}
      <Modals.Stack />
    </HaulProvider>
  );
  const { unmount } = render(content, { wrapper });
  const remount = () => {
    unmount();
    render(content, { wrapper });
  };
  return { store, remount };
};

const createRange = async (
  name: string,
  timeRange: TimeRange,
  parent?: ranger.Key,
): Promise<ranger.Range> => await client.ranges.create({ name, timeRange, parent });

/** Creates a range that started `since` ago and has no end. */
const createRunning = async (
  name: string,
  {
    parent,
    since = TimeSpan.seconds(1),
  }: { parent?: ranger.Key; since?: TimeSpan } = {},
): Promise<ranger.Range> =>
  await createRange(
    name,
    new TimeRange(TimeStamp.now().sub(since), TimeStamp.MAX),
    parent,
  );

const endRange = async (r: ranger.Range, end: TimeStamp): Promise<void> => {
  const value = { start: r.timeRange.start.valueOf(), end: end.valueOf() };
  expect(
    await client.ranges.dispatch(r.key, ranger.set({ path: ["timeRange"], value })),
  ).toBe(true);
};

/** Reads the one range with this name from the Core. */
const readRange = async (name: string): Promise<ranger.Range> => {
  const found = await reader.ranges.retrieve([name]);
  if (found.length !== 1) throw new Error(`${found.length} ranges named ${name}`);
  return found[0];
};

// The groups hold hundreds of lines on a shared Core. Role queries over them are slow
// enough to starve the toolbar's reads, so the specs find groups and lines by label.
const groupOf = (title: string): HTMLElement =>
  screen.getByLabelText(title, { selector: "section" });

/** Running shows only while a range runs, so its group can be absent. */
const queryLine = (title: string, name: string): HTMLElement | null => {
  const group = screen.queryByLabelText(title, { selector: "section" });
  return group == null
    ? null
    : within(group).queryByLabelText(name, { selector: '[role="listitem"]' });
};

const queryShowMore = (title: string): HTMLElement | null => {
  const group = screen.queryByLabelText(title, { selector: "section" });
  return group == null
    ? null
    : within(group).queryByText("Show more", { selector: "button" });
};

// The toolbar's first fetch can outlast the default 1 s wait on a busy runner.
const waitForLine = async (title: string, name: string): Promise<HTMLElement> =>
  await waitFor(
    () => {
      const line = queryLine(title, name);
      if (line == null) throw new Error(`${title} has no line named ${name}`);
      return line;
    },
    { timeout: 10_000 },
  );

/** Clicks Show more until the line loads, since other specs can fill the first page. */
const findLine = async (title: string, name: string): Promise<HTMLElement> =>
  await waitFor(
    () => {
      const line = queryLine(title, name);
      if (line != null) return line;
      const more = queryShowMore(title);
      if (more != null) fireEvent.click(more);
      throw new Error(`${title} has no line named ${name}`);
    },
    { timeout: 10_000 },
  );

const draftInput = async (): Promise<HTMLInputElement> =>
  await screen.findByPlaceholderText<HTMLInputElement>("Range name");

const typeDraft = async (name: string): Promise<void> => {
  const input = await draftInput();
  fireEvent.change(input, { target: { value: name } });
  fireEvent.keyDown(input, { key: "Enter" });
};

const isRunning = (r: ranger.Range): boolean => r.timeRange.end.equals(TimeStamp.MAX);

const expectBefore = (first: HTMLElement, second: HTMLElement): void =>
  expect(
    first.compareDocumentPosition(second) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();

const expectFocusedRange = async (store: TestStore, key: string): Promise<void> => {
  const tab = await resolveFocusedTab(store, client);
  if (tab.variant !== "resource") throw new Error("focused tab is not a resource");
  expect(tab.resource).toEqual(ranger.resourceID(key));
};

describe("range/Toolbar", () => {
  describe("groups", () => {
    it("should list a running range in Running with its elapsed time", async () => {
      const rng = await createRunning(uniqueRangeName("running"), {
        since: TimeSpan.seconds(65),
      });
      await renderToolbar();
      const line = await findLine("Running", rng.name);
      expect(line.textContent).toMatch(/1m \d\ds/);
    });

    it("should show a running range behind a page of newer ones after Show more, and keep the pages as others start", async () => {
      const oldest = await createRunning(uniqueRangeName("oldest"), {
        since: TimeSpan.minutes(5),
      });
      const newer = await client.ranges.create(
        Array.from({ length: 120 }, () => ({
          name: uniqueRangeName("newer"),
          timeRange: new TimeRange(TimeStamp.now().sub(TimeSpan.SECOND), TimeStamp.MAX),
        })),
      );
      onTestFinished(async () => await client.ranges.delete(newer.map((r) => r.key)));
      await renderToolbar();
      await waitFor(() => expect(queryShowMore("Running")).not.toBeNull(), {
        timeout: 10_000,
      });
      expect(queryLine("Running", oldest.name)).toBeNull();
      expect(within(groupOf("Running")).getByText(/^\d+\+$/)).toBeTruthy();
      expect(await findLine("Running", oldest.name)).toBeTruthy();
      // A range that starts later joins Running only through the read for its start.
      const starting = await createRange(
        uniqueRangeName("starting"),
        new TimeRange(TimeStamp.now().add(TimeSpan.SECOND), TimeStamp.MAX),
      );
      onTestFinished(async () => await client.ranges.delete(starting.key));
      await waitForLine("Running", starting.name);
      // Both lists read for the new start at once, and a busy list shows held keys.
      const list = groupOf("Running").closest<HTMLElement>('[role="list"]');
      await waitFor(() => expect(list?.getAttribute("aria-busy")).toBe("false"), {
        timeout: 10_000,
      });
      // Other specs can push this range past the loaded pages, so the count checks that
      // the pages stayed.
      const count = within(groupOf("Running")).getByText(/^\d+\+?$/).textContent ?? "";
      expect(parseInt(count, 10)).toBeGreaterThan(100);
    });

    it("should page and settle while ranges end faster than the toolbar reads", async () => {
      const oldest = await createRunning(uniqueRangeName("oldest"), {
        since: TimeSpan.minutes(5),
      });
      const create = async (
        length: number,
        timeRange: (i: number) => TimeRange,
      ): Promise<ranger.Range[]> => {
        const created = await client.ranges.create(
          Array.from({ length }, (_, i) => ({
            name: uniqueRangeName("load"),
            timeRange: timeRange(i),
          })),
        );
        onTestFinished(
          async () => await client.ranges.delete(created.map((r) => r.key)),
        );
        return created;
      };
      const now = TimeStamp.now();
      await create(120, () => new TimeRange(now.sub(TimeSpan.SECOND), TimeStamp.MAX));
      // Finished ranges fill Recent, so each read renders hundreds of lines.
      await create(
        250,
        () => new TimeRange(now.sub(TimeSpan.minutes(2)), now.sub(TimeSpan.minutes(1))),
      );
      await renderToolbar();
      await waitFor(() => expect(queryShowMore("Running")).not.toBeNull(), {
        timeout: 10_000,
      });
      // Until the spec stops it, a range ends every 25 ms, far faster than one read.
      // Batches overlap, so the ends keep coming while Show more pages.
      let producing = true;
      const ending: ranger.Range[] = [];
      const produce = async (): Promise<void> => {
        while (producing) {
          const from = TimeStamp.now();
          const batch = await create(
            40,
            (i) =>
              new TimeRange(
                from.sub(TimeSpan.SECOND),
                from.add(TimeSpan.milliseconds(250 + 25 * i)),
              ),
          );
          ending.push(...batch);
          await sleep.sleep(TimeSpan.milliseconds(500));
        }
      };
      const producer = produce();
      try {
        expect(await findLine("Running", oldest.name)).toBeTruthy();
      } finally {
        producing = false;
        await producer;
      }
      const last = ending.reduce((a, b) =>
        b.timeRange.end.after(a.timeRange.end) ? b : a,
      );
      await waitForLine("Recent", last.name);
      const list = groupOf("Recent").closest<HTMLElement>('[role="list"]');
      await waitFor(() => expect(list?.getAttribute("aria-busy")).toBe("false"), {
        timeout: 10_000,
      });
    });

    it("should move ranges from Running to Recent as they end, newest end first", async () => {
      const first = await createRunning(uniqueRangeName("first"));
      const second = await createRunning(uniqueRangeName("second"));
      await renderToolbar();
      await findLine("Running", first.name);
      await findLine("Running", second.name);
      // An end set at creation can pass before the toolbar's first fetch lands.
      await endRange(first, TimeStamp.now().add(TimeSpan.milliseconds(500)));
      await endRange(second, TimeStamp.now().add(TimeSpan.SECOND));
      const firstLine = await waitForLine("Recent", first.name);
      const secondLine = await waitForLine("Recent", second.name);
      expectBefore(secondLine, firstLine);
      expect(queryLine("Running", first.name)).toBeNull();
      expect(queryLine("Running", second.name)).toBeNull();
    });

    it("should list favorites in star order and show a hint with none", async () => {
      const a = await createRange(
        uniqueRangeName("a"),
        TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      );
      const b = await createRange(
        uniqueRangeName("b"),
        TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      );
      const { store } = await renderToolbar();
      expect(await screen.findByText("Star a range to add it here")).toBeTruthy();
      store.dispatch(Session.Range.add({ variant: "persisted", key: b.key }));
      store.dispatch(Session.Range.add({ variant: "persisted", key: a.key }));
      expectBefore(
        await findLine("Favorites", b.name),
        await findLine("Favorites", a.name),
      );
      expect(screen.queryByText("Star a range to add it here")).toBeNull();
    });

    it("should show a finished phase of a running run in Recent, then nest it once the run ends", async () => {
      const run = await createRunning(uniqueRangeName("run"), {
        since: TimeSpan.seconds(10),
      });
      const phase = await createRange(
        uniqueRangeName("phase"),
        new TimeRange(
          TimeStamp.now().sub(TimeSpan.seconds(5)),
          TimeStamp.now().add(TimeSpan.SECOND),
        ),
        run.key,
      );
      await renderToolbar();
      const finished = await waitForLine("Recent", phase.name);
      expect(within(finished).getByText(run.name)).toBeTruthy();
      // "should stop a running range" covers Stop, so the run ends through the client.
      expect(
        await client.ranges.dispatch(
          run.key,
          ranger.set({
            path: ["timeRange"],
            value: new TimeRange(run.timeRange.start, TimeStamp.now()).numeric,
          }),
        ),
      ).toBe(true);
      const ended = await findLine("Recent", run.name);
      await waitFor(() => expect(queryLine("Recent", phase.name)).toBeNull());
      fireEvent.click(within(ended).getByRole("button", { name: "Expand" }));
      expect(await findLine("Recent", phase.name)).toBeTruthy();
    });

    it("should keep a collapsed group collapsed after a remount", async () => {
      const { remount } = await renderToolbar();
      const header = async (): Promise<HTMLElement> =>
        await screen.findByRole("button", { name: "Recent" });
      fireEvent.click(await header());
      await waitFor(async () =>
        expect((await header()).getAttribute("aria-expanded")).toBe("false"),
      );
      remount();
      expect((await header()).getAttribute("aria-expanded")).toBe("false");
    });
  });

  describe("start", () => {
    it("should start a range at the click time on Enter", async () => {
      const name = uniqueRangeName("started");
      await renderToolbar();
      const before = TimeStamp.now();
      fireEvent.click(await screen.findByRole("button", { name: "Start range" }));
      const after = TimeStamp.now();
      await typeDraft(name);
      const rng = await waitFor(async () => await readRange(name));
      expect(isRunning(rng)).toBe(true);
      expect(rng.timeRange.start.beforeEq(after)).toBe(true);
      expect(rng.timeRange.start.afterEq(before)).toBe(true);
      expect(await findLine("Running", name)).toBeTruthy();
    });

    it("should report a started range to analytics", async () => {
      const analytics = createTestSink();
      await renderToolbar({ analytics });
      fireEvent.click(await screen.findByRole("button", { name: "Start range" }));
      await typeDraft(uniqueRangeName("reported"));
      await waitFor(() =>
        expect(analytics.capture).toHaveBeenCalledWith("range_created", {}),
      );
    });

    it("should close the draft and start nothing on Escape", async () => {
      const name = uniqueRangeName("canceled");
      await renderToolbar();
      fireEvent.click(await screen.findByRole("button", { name: "Start range" }));
      const input = await draftInput();
      fireEvent.change(input, { target: { value: name } });
      fireEvent.keyDown(input, { key: "Escape" });
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Range name")).toBeNull(),
      );
      expect(await reader.ranges.retrieve([name])).toEqual([]);
    });

    it("should start a child range under the range", async () => {
      const parent = await createRunning(uniqueRangeName("parent"));
      const name = uniqueRangeName("child");
      await renderToolbar();
      const line = await findLine("Running", parent.name);
      fireEvent.click(
        await within(line).findByRole("button", { name: "Start child range" }),
      );
      await typeDraft(name);
      const child = await waitFor(async () => await readRange(name));
      expect(child.parent).toBe(parent.key);
      expect(isRunning(child)).toBe(true);
    });

    it("should start a child range from the menu of a running range", async () => {
      const parent = await createRunning(uniqueRangeName("parent"));
      const name = uniqueRangeName("child");
      await renderToolbar();
      const line = await findLine("Running", parent.name);
      await openContextMenu(() => line);
      fireEvent.click(
        await screen.findByRole("menuitem", { name: "Start child range" }),
      );
      await typeDraft(name);
      const child = await waitFor(async () => await readRange(name));
      expect(child.parent).toBe(parent.key);
    });

    it("should open the create form for a child of a range that is not running", async () => {
      const rng = await createRange(
        uniqueRangeName("ended"),
        new TimeRange(TimeStamp.now().sub(TimeSpan.minutes(2)), TimeStamp.now()),
      );
      await renderToolbar({ favorites: [rng] });
      const line = await findLine("Favorites", rng.name);
      await openContextMenu(() => line);
      fireEvent.click(
        await screen.findByRole("menuitem", { name: "Start child range" }),
      );
      expect(await screen.findByRole("button", { name: "Create" })).toBeTruthy();
    });

    it("should open a collapsed Running group to show the draft", async () => {
      const { store } = await renderToolbar();
      store.dispatch(
        Session.Range.setGroupCollapsed({ group: "running", collapsed: true }),
      );
      fireEvent.click(await screen.findByRole("button", { name: "Start range" }));
      expect(await draftInput()).toBeTruthy();
    });
  });

  describe("next", () => {
    it("should stop a range and start the next one with the suggested name", async () => {
      const token = uniqueRangeName("run");
      const campaign = await createRunning(uniqueRangeName("campaign"));
      const run = await createRunning(`${token} 7`, { parent: campaign.key });
      await renderToolbar();
      const line = await findLine("Running", run.name);
      fireEvent.click(
        await within(line).findByRole("button", { name: "Stop and start next" }),
      );
      const input = await draftInput();
      await waitFor(() => expect(input.value).toBe(`${token} 8`));
      fireEvent.keyDown(input, { key: "Enter" });
      const next = await waitFor(async () => await readRange(`${token} 8`));
      const stopped = await waitFor(async () => {
        const r = await readRange(run.name);
        expect(isRunning(r)).toBe(false);
        return r;
      });
      expect(next.parent).toBe(campaign.key);
      expect(isRunning(next)).toBe(true);
      expect(stopped.timeRange.end.valueOf()).toBe(next.timeRange.start.valueOf());
    });

    it("should stop the running ranges below the range too", async () => {
      const campaign = await createRunning(uniqueRangeName("campaign"));
      const run = await createRunning(uniqueRangeName("run"), { parent: campaign.key });
      const phase = await createRunning(uniqueRangeName("phase"), { parent: run.key });
      const name = uniqueRangeName("next");
      await renderToolbar();
      const line = await findLine("Running", run.name);
      fireEvent.click(
        await within(line).findByRole("button", { name: "Stop and start next" }),
      );
      await typeDraft(name);
      const next = await waitFor(async () => await readRange(name));
      await waitFor(async () => {
        const [r, p] = await Promise.all([readRange(run.name), readRange(phase.name)]);
        expect(r.timeRange.end.valueOf()).toBe(next.timeRange.start.valueOf());
        expect(p.timeRange.end.valueOf()).toBe(next.timeRange.start.valueOf());
      });
    });

    /** Creates a running run under a running campaign and shows it in Running. */
    const renderRun = async (name = uniqueRangeName("run")) => {
      const campaign = await createRunning(uniqueRangeName("campaign"));
      const run = await createRunning(name, { parent: campaign.key });
      await renderToolbar();
      return { campaign, run, line: await findLine("Running", run.name) };
    };

    it("should start the next range from the menu", async () => {
      const { campaign, run, line } = await renderRun();
      const name = uniqueRangeName("next");
      await openContextMenu(() => line);
      fireEvent.click(await screen.findByRole("menuitem", { name: "Next" }));
      await typeDraft(name);
      const next = await waitFor(async () => await readRange(name));
      expect(next.parent).toBe(campaign.key);
      await waitFor(async () =>
        expect((await readRange(run.name)).timeRange.end.valueOf()).toBe(
          next.timeRange.start.valueOf(),
        ),
      );
    });

    it("should leave the range running when Escape closes the draft", async () => {
      const { run, line } = await renderRun();
      fireEvent.click(
        await within(line).findByRole("button", { name: "Stop and start next" }),
      );
      fireEvent.keyDown(await draftInput(), { key: "Escape" });
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Range name")).toBeNull(),
      );
      expect(isRunning(await readRange(run.name))).toBe(true);
    });

    it("should close the draft when its range stops", async () => {
      const { line } = await renderRun();
      fireEvent.click(
        await within(line).findByRole("button", { name: "Stop and start next" }),
      );
      await draftInput();
      fireEvent.click(within(line).getByRole("button", { name: "Stop" }));
      await waitFor(
        () => expect(screen.queryByPlaceholderText("Range name")).toBeNull(),
        { timeout: 5_000 },
      );
    });

    it("should leave the range running when the next one fails to start", async () => {
      const campaign = await createRunning(uniqueRangeName("campaign"));
      const run = await createRunning(uniqueRangeName("run"), { parent: campaign.key });
      const subject = await createTestClientWithGrants(client, {
        create: [ranger.TYPE_RESOURCE_ID],
        update: [ranger.TYPE_RESOURCE_ID],
        retrieve: [
          ranger.TYPE_RESOURCE_ID,
          label.TYPE_RESOURCE_ID,
          project.TYPE_RESOURCE_ID,
          panel.TYPE_RESOURCE_ID,
        ],
      });
      await renderToolbar({ as: subject });
      const line = await findLine("Running", run.name);
      fireEvent.click(
        await within(line).findByRole("button", { name: "Stop and start next" }),
      );
      const input = await draftInput();
      const { role } = await subject.users.current.retrieve();
      const policies = await client.access.policies.retrieve({
        where: { roles: [role] },
      });
      await client.access.policies.delete(
        policies.filter((p) => p.actions.includes("create")).map((p) => p.key),
      );
      const name = uniqueRangeName("refused");
      fireEvent.change(input, { target: { value: name } });
      fireEvent.keyDown(input, { key: "Enter" });
      // Long enough for the refused create to answer and for any stop to land.
      await sleep.sleep(TimeSpan.milliseconds(500));
      expect(await reader.ranges.retrieve([name])).toEqual([]);
      expect(isRunning(await readRange(run.name))).toBe(true);
    });

    it("should keep a newer draft when an earlier suggestion answers late", async () => {
      const token = uniqueRangeName("run");
      const { line } = await renderRun(`${token} 7`);
      fireEvent.click(
        await within(line).findByRole("button", { name: "Stop and start next" }),
      );
      fireEvent.click(screen.getByRole("button", { name: "Start range" }));
      const input = await draftInput();
      // Long enough for the suggestion of the first draft to answer.
      await sleep.sleep(TimeSpan.milliseconds(500));
      expect(input.value).toBe("");
      expect(screen.getByPlaceholderText<HTMLInputElement>("Range name")).toBe(input);
    });
  });

  describe("line", () => {
    it("should stop a running range", async () => {
      const rng = await createRunning(uniqueRangeName("stop"));
      await renderToolbar();
      const line = await findLine("Running", rng.name);
      fireEvent.click(await within(line).findByRole("button", { name: "Stop" }));
      await waitFor(async () =>
        expect(isRunning(await readRange(rng.name))).toBe(false),
      );
      await findLine("Recent", rng.name);
    });

    it("should remove a favorite with its star", async () => {
      const rng = await createRange(
        uniqueRangeName("star"),
        TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      );
      const { store } = await renderToolbar({ favorites: [rng] });
      const line = await findLine("Favorites", rng.name);
      fireEvent.click(within(line).getByRole("checkbox", { name: "Favorite" }));
      await waitFor(() => expect(queryLine("Favorites", rng.name)).toBeNull());
      expect(store.getState()[Session.Range.SLICE_NAME].ranges).toEqual([]);
    });

    it("should rename the line that the menu opened over", async () => {
      const rng = await createRunning(uniqueRangeName("rename"));
      await renderToolbar({ favorites: [rng] });
      await findLine("Running", rng.name);
      const line = await findLine("Favorites", rng.name);
      await openContextMenu(() => line);
      fireEvent.click(await screen.findByRole("menuitem", { name: "Rename" }));
      const nameOf = (el: HTMLElement): HTMLElement | null =>
        el.querySelector(`[id="${List.itemNameID(rng.key)}"]`);
      await waitFor(() =>
        expect(nameOf(line)?.getAttribute("contenteditable")).toBe("true"),
      );
      const running = await findLine("Running", rng.name);
      expect(nameOf(running)?.getAttribute("contenteditable")).not.toBe("true");
    });

    it("should open the overview of the range on a click", async () => {
      const rng = await createRange(
        uniqueRangeName("open"),
        TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      );
      const { store } = await renderToolbar({ favorites: [rng] });
      fireEvent.click(await findLine("Favorites", rng.name));
      await expectFocusedRange(store, rng.key);
    });
  });

  describe("header", () => {
    it("should open the create form from Create past range", async () => {
      await renderToolbar();
      fireEvent.click(await screen.findByRole("button", { name: "Create past range" }));
      expect(await screen.findByRole("button", { name: "Create" })).toBeTruthy();
    });

    it("should open the range explorer", async () => {
      const { store } = await renderToolbar();
      fireEvent.click(
        await screen.findByRole("button", { name: "Open range explorer" }),
      );
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "view") throw new Error("focused tab is not a view");
      expect(tab.type).toBe(Range.Explorer.TAB_TYPE);
    });
  });

  describe("drag", () => {
    it("should picture a dragged line as its range", async () => {
      const rng = await createRange(
        uniqueRangeName("picture"),
        TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      );
      await renderToolbar({ favorites: [rng] });
      fireEvent.dragStart(await findLine("Favorites", rng.name));
      const preview = document.querySelector(".lyra-haul__layer .lyra-haul-preview");
      expect(preview?.querySelector(".lyra-haul-preview__name")?.textContent).toBe(
        rng.name,
      );
      expect(preview?.querySelector(".lyra-icon--range")).toBeInstanceOf(Element);
    });

    it("should add a range dropped on the range picker of a line plot", async () => {
      const rng = await createRange(
        uniqueRangeName("drag"),
        TimeStamp.now().spanRange(TimeSpan.seconds(10)),
      );
      const { key } = await renderLinePlot(
        () => (
          <HaulProvider>
            <LinePlotToolbar />
            {Range.TOOLBAR.content}
          </HaulProvider>
        ),
        {
          preloadedState: () => ({
            [Session.Range.SLICE_NAME]: {
              ...Session.Range.ZERO_SLICE_STATE,
              ranges: [{ variant: "persisted", key: rng.key }],
            },
          }),
        },
      );
      fireEvent.dragStart(await findLine("Favorites", rng.name));
      fireEvent.drop(await screen.findByLabelText("Ranges"));
      await waitFor(async () =>
        expect((await reader.lineplots.retrieve(key)).ranges).toContainEqual({
          variant: "key",
          key: rng.key,
        }),
      );
    });
  });

  describe("permissions", () => {
    const createSubject = async (grants: Grants): Promise<Client> =>
      await createTestClientWithGrants(client, {
        ...grants,
        retrieve: [
          ranger.TYPE_RESOURCE_ID,
          label.TYPE_RESOURCE_ID,
          project.TYPE_RESOURCE_ID,
          panel.TYPE_RESOURCE_ID,
        ],
      });

    const renderRun = async (grants: Grants): Promise<HTMLElement> => {
      const campaign = await createRunning(uniqueRangeName("campaign"));
      const run = await createRunning(uniqueRangeName("run"), { parent: campaign.key });
      await renderToolbar({ as: await createSubject(grants) });
      // The same policy read answers every grant, so the explorer button shows that it
      // landed.
      await screen.findByRole("button", { name: "Open range explorer" });
      return await findLine("Running", run.name);
    };

    const WRITES = ["Start child range", "Stop and start next", "Stop"];

    it("should hide the range writes from a user with no range grants", async () => {
      const line = await renderRun({});
      for (const name of WRITES)
        expect(within(line).queryByRole("button", { name })).toBeNull();
      expect(screen.queryByRole("button", { name: "Start range" })).toBeNull();
      expect(screen.queryByRole("button", { name: "Create past range" })).toBeNull();
    });

    it("should show the range writes to a user granted them", async () => {
      const line = await renderRun({
        create: [ranger.TYPE_RESOURCE_ID],
        update: [ranger.TYPE_RESOURCE_ID],
      });
      for (const name of WRITES)
        expect(within(line).getByRole("button", { name })).toBeTruthy();
      expect(screen.getByRole("button", { name: "Start range" })).toBeTruthy();
      expect(screen.getByRole("button", { name: "Create past range" })).toBeTruthy();
    });
  });
});
