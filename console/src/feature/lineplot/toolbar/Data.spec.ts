// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type lineplot, type ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import {
  dynamicWindow,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  onTestFinished,
  vi,
} from "vitest";

import { client, renderLinePlot } from "@/feature/lineplot/testutil";
import { Toolbar } from "@/feature/lineplot/toolbar";
import { createTestRange } from "@/platform/range/testutil";
import {
  findDialogTriggerByText,
  findTagCloseButton,
  getInputItem,
  uniqueToken,
} from "@/testutil";

// A separate client reads what the Core saved, not the rendering client's cache.
const reader = createTestClient();

const renderData = async (ranges?: lineplot.New["ranges"]) => {
  const handle = await renderLinePlot(Toolbar, {
    linePlot: ranges === undefined ? {} : { ranges },
  });
  await screen.findByText("Ranges");
  return handle;
};

const rolling = (span: TimeSpan): ranger.Ref => ({
  variant: "window",
  window: dynamicWindow(span),
});

const label = (span: TimeSpan): string => span.toString();

const SEARCH_PLACEHOLDER = "Search ranges or type a duration or time";

const openSearch = (): HTMLElement => {
  fireEvent.click(screen.getByRole("button", { name: "Ranges" }));
  return screen.getByPlaceholderText(SEARCH_PLACEHOLDER);
};

const search = (term: string): void => {
  fireEvent.change(openSearch(), { target: { value: term } });
};

// A duration also searches the Core's ranges, so the live option is found by its
// text.
const liveOption = (): HTMLElement =>
  screen.getByRole("option", { name: /Live window/ });

// The rolling window's tag shows its span too, so a preset is told apart by its toggle.
const clickPreset = (span: TimeSpan): void => {
  const preset = screen
    .getAllByRole("button", { name: label(span) })
    .find((b) => b.hasAttribute("aria-pressed"));
  assertDefined(preset);
  fireEvent.click(preset);
};

// Creates a range that starts the given hours before the end of time, so only a planned
// range sorts ahead of it in the newest-first list. It leads the lists of other specs
// too, so it is deleted once the test finishes.
const createLeadingRange = async (hours: number): Promise<ranger.Range> => {
  const start = TimeStamp.MAX.sub(TimeSpan.hours(hours));
  const rng = await createTestRange(client, {
    timeRange: start.spanRange(TimeSpan.hours(1)),
  });
  onTestFinished(async () => await client.ranges.delete(rng.key));
  return rng;
};

// Reads the option texts once they show every given range. The ranges of other specs
// can move the list only before the read.
const optionsShowing = async (...ranges: ranger.Range[]): Promise<string[]> =>
  await waitFor(() => {
    const texts = screen.getAllByRole("option").map((o) => o.textContent ?? "");
    for (const { name } of ranges)
      expect(texts.some((t) => t.includes(name))).toBe(true);
    return texts;
  });

const expectRanges = async (key: string, ranges: ranger.Ref[]): Promise<void> =>
  await waitFor(async () =>
    expect((await reader.lineplots.retrieve(key)).ranges).toEqual(ranges),
  );

const local = (hour: number): TimeStamp =>
  new TimeStamp(new Date(2026, 7, 23, hour, 0));

const fixed = (startHour: number, endHour: number): ranger.Ref => ({
  variant: "window",
  window: staticWindow(new TimeRange(local(startHour), local(endHour))),
});

const FIVE_MIN = TimeSpan.minutes(5);
const FORTY_FIVE_MIN = TimeSpan.minutes(45);

describe("lineplot/toolbar/Data", () => {
  describe("rolling window", () => {
    it("lists the presets as a row, apart from the ranges", async () => {
      await renderData([]);
      openSearch();
      expect(screen.getByRole("button", { name: label(FIVE_MIN) })).toBeDefined();
      expect(screen.queryAllByRole("option")).toHaveLength(0);
    });

    it("keeps one window, replacing it with the next preset picked", async () => {
      const { key } = await renderData([]);
      openSearch();
      clickPreset(FIVE_MIN);
      await expectRanges(key, [rolling(FIVE_MIN)]);
      clickPreset(TimeSpan.hours(1));
      await expectRanges(key, [rolling(TimeSpan.hours(1))]);
    });

    it("keeps the other ranges when the window changes", async () => {
      const created = await createTestRange(client);
      const persisted: ranger.Ref = { variant: "key", key: created.key };
      const { key } = await renderData([persisted, rolling(FIVE_MIN)]);
      openSearch();
      clickPreset(TimeSpan.hours(1));
      await expectRanges(key, [persisted, rolling(TimeSpan.hours(1))]);
    });

    it("clears the window when its preset is clicked again", async () => {
      const { key } = await renderData([rolling(FIVE_MIN)]);
      openSearch();
      clickPreset(FIVE_MIN);
      await expectRanges(key, []);
    });

    it("clears the window with the Off option", async () => {
      const { key } = await renderData([rolling(FIVE_MIN)]);
      openSearch();
      fireEvent.click(screen.getByRole("button", { name: "Off" }));
      await expectRanges(key, []);
    });

    it("marks Off while the plot has no window", async () => {
      await renderData([]);
      openSearch();
      const off = screen.getByRole("button", { name: "Off" });
      expect(off.getAttribute("aria-pressed")).toBe("true");
    });

    it("sets the window to a duration typed into the search on Enter", async () => {
      const { key } = await renderData([rolling(FIVE_MIN)]);
      const input = openSearch();
      fireEvent.change(input, { target: { value: "45m" } });
      fireEvent.keyDown(input, { key: "Enter" });
      await expectRanges(key, [rolling(FORTY_FIVE_MIN)]);
      expect(screen.queryByPlaceholderText(SEARCH_PLACEHOLDER)).toBeNull();
    });

    it("keeps Enter on a typed duration from selecting a range", async () => {
      const range = await client.ranges.create({
        name: uniqueName("45m burn"),
        timeRange: new TimeRange(1_000_000_000, 3_000_000_000),
      });
      const { key } = await renderData([rolling(FIVE_MIN)]);
      const input = openSearch();
      fireEvent.change(input, { target: { value: "45m" } });
      await screen.findByText(range.name);
      act(() => {
        fireEvent.keyDown(input, { key: "Enter", code: "Enter" });
      });
      await expectRanges(key, [rolling(FORTY_FIVE_MIN)]);
    });

    it("lists a live option for a duration typed into the search", async () => {
      await renderData([]);
      search("45m");
      const [first] = screen.getAllByRole("option");
      expect(first).toBe(liveOption());
      expect(within(first).getByText(label(FORTY_FIVE_MIN))).toBeDefined();
      expect(screen.queryByText("No ranges found")).toBeNull();
    });

    it("lists a live option for a span after last or past", async () => {
      await renderData([]);
      const input = openSearch();
      fireEvent.change(input, { target: { value: "last 2 hours" } });
      expect(within(liveOption()).getByText(label(TimeSpan.hours(2)))).toBeDefined();
      fireEvent.change(input, { target: { value: "the past hour" } });
      expect(within(liveOption()).getByText(label(TimeSpan.HOUR))).toBeDefined();
    });

    it("says no ranges match a search that finds none", async () => {
      await renderData([]);
      search(uniqueToken());
      await screen.findByText("No ranges found");
    });

    it("replaces the window with the live option picked", async () => {
      const { key } = await renderData([fixed(1, 2), rolling(FIVE_MIN)]);
      search("45m");
      fireEvent.click(liveOption());
      await expectRanges(key, [fixed(1, 2), rolling(FORTY_FIVE_MIN)]);
    });

    it("clears the window when its live option is picked again", async () => {
      const { key } = await renderData([rolling(FIVE_MIN)]);
      search("5m");
      fireEvent.click(liveOption());
      await expectRanges(key, []);
    });

    it("marks the live option selected while its window is live", async () => {
      await renderData([rolling(FIVE_MIN)]);
      search("5m");
      expect(liveOption().getAttribute("aria-selected")).toBe("true");
    });

    it("ignores Enter when the search is not a duration", async () => {
      const { key } = await renderData([rolling(FIVE_MIN)]);
      const input = openSearch();
      for (const value of ["0s", "banana"]) {
        fireEvent.change(input, { target: { value } });
        fireEvent.keyDown(input, { key: "Enter" });
      }
      expect(
        await screen.findByText(label(FIVE_MIN), { selector: "span" }),
      ).toBeDefined();
      expect(screen.getByPlaceholderText(SEARCH_PLACEHOLDER)).toBe(input);
      expect((await reader.lineplots.retrieve(key)).ranges).toEqual([
        rolling(FIVE_MIN),
      ]);
    });

    it("keeps the presets out of the search list's keys", async () => {
      await renderData([]);
      search("banana");
      act(() => {
        fireEvent.keyDown(window, { code: "ArrowDown" });
        fireEvent.keyDown(window, { code: "Enter" });
      });
      const pressed = screen
        .getAllByRole("button")
        .filter((b) => b.getAttribute("aria-pressed") === "true")
        .map((b) => b.textContent);
      expect(pressed).toEqual(["Off"]);
    });

    it("clears the window when its tag is closed", async () => {
      const { key } = await renderData([rolling(FORTY_FIVE_MIN)]);
      await screen.findByText(label(FORTY_FIVE_MIN));
      fireEvent.click(findTagCloseButton(label(FORTY_FIVE_MIN)));
      await expectRanges(key, []);
    });
  });

  describe("ranges", () => {
    it("adds a range that no one starred as a key reference", async () => {
      const created = await createTestRange(client);
      const { key } = await renderData([]);
      search(created.name);
      // Before the search answers, the ranges of other specs can move the option out of
      // the list. The spec clicks it once the list holds only the answer.
      const option = await waitFor(() => {
        const [only, ...rest] = screen.getAllByRole("option");
        expect(rest).toHaveLength(0);
        expect(only.textContent).toContain(created.name);
        return only;
      });
      fireEvent.click(option);
      await expectRanges(key, [{ variant: "key", key: created.key }]);
    });

    it("lists the ranges newest first", async () => {
      // The picker lists the ranges of other specs too, so the spec checks the order
      // of every listed row. Its own two ranges make sure there are at least two.
      await createTestRange(client);
      await createTestRange(client);
      await renderData([]);
      openSearch();
      await waitFor(async () => {
        const keys = screen.getAllByRole("option").map((o) => o.id);
        // Another spec can delete a listed range, and the Core returns only the ranges
        // that exist.
        const ranges = await reader.ranges.retrieve({ where: { key: { in: keys } } });
        const starts = keys.flatMap((k) => {
          const range = ranges.find((r) => r.key === k);
          return range == null ? [] : [range.timeRange.start];
        });
        expect(starts.length).toBeGreaterThanOrEqual(2);
        starts
          .slice(1)
          .forEach((start, i) =>
            expect(start.valueOf()).toBeLessThanOrEqual(starts[i].valueOf()),
          );
      });
    });

    it("lists the plotted ranges first", async () => {
      const plotted = await createTestRange(client);
      // It sorts ahead of the plotted range, which leads only because it is plotted.
      const newer = await createLeadingRange(2);
      await renderData([{ variant: "key", key: plotted.key }]);
      // The tag reads the range, so the picker can put it first when it opens.
      await screen.findByText(plotted.name);
      openSearch();
      const [first] = await optionsShowing(plotted, newer);
      expect(first).toContain(plotted.name);
    });

    it("marks each key reference's tag with the stage of its range", async () => {
      const now = TimeStamp.now();
      const create = async (start: TimeSpan, end: TimeSpan): Promise<ranger.Ref> => {
        const { key } = await client.ranges.create({
          name: uniqueName("range"),
          timeRange: new TimeRange(now.add(start), now.add(end)),
        });
        return { variant: "key", key };
      };
      await renderData([
        await create(TimeSpan.hours(-2), TimeSpan.hours(-1)),
        await create(TimeSpan.hours(-1), TimeSpan.hours(1)),
        await create(TimeSpan.hours(1), TimeSpan.hours(2)),
      ]);
      expect(await screen.findByLabelText("Completed")).toBeDefined();
      expect(screen.getByLabelText("In progress")).toBeDefined();
      expect(screen.getByLabelText("To do")).toBeDefined();
    });

    it("names a key reference's tag after the Core range", async () => {
      const created = await createTestRange(client);
      await renderData([{ variant: "key", key: created.key }]);
      expect(await screen.findByText(created.name)).toBeDefined();
    });
  });

  it("marks a range the Core deleted and removes it when its tag is closed", async () => {
    const created = await createTestRange(client);
    await client.ranges.delete(created.key);
    const { key } = await renderData([{ variant: "key", key: created.key }]);
    await screen.findByText("Deleted range");
    fireEvent.click(findTagCloseButton("Deleted range"));
    await expectRanges(key, []);
  });

  describe("typed time range", () => {
    // The day words depend on today, so the clock is fixed.
    beforeEach(() => {
      vi.useFakeTimers({ toFake: ["Date", "Temporal"] });
      vi.setSystemTime(new Date(2026, 10, 3, 15, 0));
    });
    afterEach(() => vi.useRealTimers());

    const YESTERDAY_ONE_TO_TWO = "Yesterday from 1pm to 2pm";

    const fixedOption = (): Promise<HTMLElement> =>
      screen.findByRole("option", { name: /Fixed window/ });

    it("lists a fixed option labeled with the times it reads", async () => {
      await renderData([]);
      search(YESTERDAY_ONE_TO_TWO);
      const option = await fixedOption();
      expect(within(option).getByText("Yesterday 13:00 → 14:00")).toBeDefined();
      expect(screen.getAllByRole("option")[0]).toBe(option);
    });

    it("adds the fixed window beside the live one on Enter", async () => {
      const { key } = await renderData([rolling(FIVE_MIN)]);
      const input = openSearch();
      fireEvent.change(input, { target: { value: YESTERDAY_ONE_TO_TWO } });
      await fixedOption();
      fireEvent.keyDown(input, { key: "Enter" });
      const start = new TimeStamp(new Date(2026, 10, 2, 13, 0));
      await expectRanges(key, [
        rolling(FIVE_MIN),
        {
          variant: "window",
          window: staticWindow(new TimeRange(start, start.add(TimeSpan.HOUR))),
        },
      ]);
      expect(screen.queryByPlaceholderText(SEARCH_PLACEHOLDER)).toBeNull();
    });

    it("adds the fixed window picked from the list", async () => {
      const { key } = await renderData([]);
      search(YESTERDAY_ONE_TO_TWO);
      fireEvent.click(await fixedOption());
      const start = new TimeStamp(new Date(2026, 10, 2, 13, 0));
      await expectRanges(key, [
        {
          variant: "window",
          window: staticWindow(new TimeRange(start, start.add(TimeSpan.HOUR))),
        },
      ]);
    });

    it("adds the hovered range on Enter instead of the fixed window", async () => {
      const range = await client.ranges.create({
        name: uniqueName(YESTERDAY_ONE_TO_TWO),
        timeRange: new TimeRange(1_000_000_000, 3_000_000_000),
      });
      const { key } = await renderData([]);
      const input = openSearch();
      fireEvent.change(input, { target: { value: YESTERDAY_ONE_TO_TWO } });
      await fixedOption();
      const rangeOption = (await screen.findByText(range.name)).closest(
        "[role=option]",
      );
      assertDefined(rangeOption);
      fireEvent.pointerMove(rangeOption, { pointerType: "mouse", clientX: 1 });
      fireEvent.pointerMove(rangeOption, { pointerType: "mouse", clientX: 2 });
      expect(rangeOption.classList.contains("lyra--hovered")).toBe(true);
      act(() => {
        fireEvent.keyDown(input, { key: "Enter", code: "Enter" });
      });
      await expectRanges(key, [{ variant: "key", key: range.key }]);
    });

    it("keeps last week a fixed calendar week", async () => {
      await renderData([]);
      search("last week");
      await fixedOption();
      expect(screen.queryByRole("option", { name: /Live window/ })).toBeNull();
    });

    it("lists no fixed option when other words name a range", async () => {
      await renderData([]);
      const input = openSearch();
      fireEvent.change(input, { target: { value: YESTERDAY_ONE_TO_TWO } });
      await fixedOption();
      fireEvent.change(input, { target: { value: `monday ${uniqueToken()}` } });
      await screen.findByText("No ranges found");
      expect(screen.queryByRole("option", { name: /Fixed window/ })).toBeNull();
    });
  });

  describe("static window", () => {
    it("marks each tag with the stage of its window", async () => {
      const now = TimeStamp.now();
      const window = (start: TimeSpan, end: TimeSpan): ranger.Ref => ({
        variant: "window",
        window: staticWindow(new TimeRange(now.add(start), now.add(end))),
      });
      await renderData([
        window(TimeSpan.hours(-2), TimeSpan.hours(-1)),
        window(TimeSpan.hours(-1), TimeSpan.hours(1)),
        window(TimeSpan.hours(1), TimeSpan.hours(2)),
      ]);
      expect(await screen.findByLabelText("Completed")).toBeDefined();
      expect(screen.getByLabelText("In progress")).toBeDefined();
      expect(screen.getByLabelText("To do")).toBeDefined();
    });

    describe("label", () => {
      // The day names depend on today, so the clock is fixed.
      beforeEach(() => {
        vi.useFakeTimers({ toFake: ["Date", "Temporal"] });
        vi.setSystemTime(new Date(2026, 10, 3, 15, 0));
      });
      afterEach(() => vi.useRealTimers());

      it("labels its tag with the times, read-only", async () => {
        await renderData([fixed(14, 16)]);
        expect(await screen.findByText("Aug 23 14:00 → 16:00")).toBeDefined();
        expect(screen.queryByLabelText("Start")).toBeNull();
      });

      it("shows seconds and no day for a short window today", async () => {
        const noon = new Date();
        noon.setHours(12, 0, 5, 0);
        const start = new TimeStamp(noon);
        await renderData([
          {
            variant: "window",
            window: staticWindow(new TimeRange(start, start.add(TimeSpan.seconds(30)))),
          },
        ]);
        expect(await screen.findByText("12:00:05 → 12:00:35")).toBeDefined();
      });

      it("shows the end's day when the window crosses midnight", async () => {
        await renderData([
          {
            variant: "window",
            window: staticWindow(
              new TimeRange(local(22), new TimeStamp(new Date(2026, 7, 24, 2, 0))),
            ),
          },
        ]);
        expect(await screen.findByText("Aug 23 22:00 → Aug 24 02:00")).toBeDefined();
      });
    });
  });

  it("removes a static window when its tag is closed", async () => {
    const { key } = await renderData([
      {
        variant: "window",
        window: staticWindow(new TimeRange(1_000_000_000, 2_000_000_000)),
      },
    ]);
    fireEvent.click(await screen.findByRole("button", { name: /^Remove / }));
    await expectRanges(key, []);
  });

  it("clears the X channel with the clear button on its icon", async () => {
    const index = await client.channels.create({
      name: uniqueName("x_index"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const { key } = await renderLinePlot(Toolbar, {
      linePlot: { channels: { x1: index.key } },
    });
    fireEvent.click(await screen.findByLabelText(`Clear ${index.name}`));
    await waitFor(async () => {
      const plot = await reader.lineplots.retrieve(key);
      expect(plot.channels.x1).toBe(0);
    });
    expect(screen.queryByLabelText(`Clear ${index.name}`)).toBeNull();
  });

  it("shows an empty X channel as time with an auto chip", async () => {
    await renderData();
    const item = getInputItem("X1");
    expect(within(item).getByText("Time")).toBeDefined();
    expect(within(item).getByText("Auto")).toBeDefined();
  });

  it("returns the X channel to time with the pinned time option", async () => {
    const index = await client.channels.create({
      name: uniqueName("x_index"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const { key } = await renderLinePlot(Toolbar, {
      linePlot: { channels: { x1: index.key } },
    });
    fireEvent.click(await findDialogTriggerByText(index.name));
    fireEvent.click(
      await screen.findByText(
        "Automatically uses each channel's time. Best for most needs.",
      ),
    );
    await waitFor(async () => {
      const plot = await reader.lineplots.retrieve(key);
      expect(plot.channels.x1).toBe(0);
    });
  });
});
