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
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { client, renderLinePlot } from "@/feature/lineplot/testutil";
import { Toolbar } from "@/feature/lineplot/toolbar";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { assertDefined, findTagCloseButton, uniqueName } from "@/testutil";

// A separate client reads what the Core saved, not the rendering client's cache.
const reader = createTestClient();

const renderData = async (
  ranges?: lineplot.New["ranges"],
  favorites: Session.Range.State[] = [],
) => {
  const handle = await renderLinePlot(Toolbar, {
    linePlot: ranges === undefined ? {} : { ranges },
    preloadedState: () => ({
      [Session.Range.SLICE_NAME]: {
        ...Session.Range.ZERO_SLICE_STATE,
        ranges: favorites,
      },
    }),
  });
  await screen.findByText("Ranges");
  return handle;
};

const rolling = (span: TimeSpan): ranger.Ref => ({
  variant: "window",
  window: dynamicWindow(span),
});

const label = (span: TimeSpan): string => span.toString();

const openSearch = (): HTMLElement => {
  fireEvent.click(screen.getByRole("button", { name: "Ranges" }));
  return screen.getByPlaceholderText("Search favorites or type a duration");
};

const search = (term: string): void => {
  fireEvent.change(openSearch(), { target: { value: term } });
};

// The rolling window's tag shows its span too, so a preset is told apart by its toggle.
const clickPreset = (span: TimeSpan): void => {
  const preset = screen
    .getAllByRole("button", { name: label(span) })
    .find((b) => b.hasAttribute("aria-pressed"));
  assertDefined(preset);
  fireEvent.click(preset);
};

const expectRanges = async (key: string, ranges: ranger.Ref[]): Promise<void> =>
  await waitFor(async () =>
    expect((await reader.lineplots.retrieve(key)).ranges).toEqual(ranges),
  );

const editTime = (label: string, text: string): void => {
  fireEvent.click(screen.getByLabelText(label));
  const field = screen.getByRole("textbox");
  fireEvent.change(field, { target: { value: text } });
  fireEvent.keyDown(field, { key: "Enter" });
};

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
    it("lists the presets as a row, apart from the favorites", async () => {
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

    it("sets the window to a duration typed into the search on Enter", async () => {
      const { key } = await renderData([rolling(FIVE_MIN)]);
      const input = openSearch();
      fireEvent.change(input, { target: { value: "45m" } });
      fireEvent.keyDown(input, { key: "Enter" });
      await expectRanges(key, [rolling(FORTY_FIVE_MIN)]);
    });

    it("lists no option for a duration typed into the search", async () => {
      await renderData([]);
      search("45m");
      expect(screen.queryAllByRole("option")).toHaveLength(0);
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
        .filter((b) => b.getAttribute("aria-pressed") === "true");
      expect(pressed).toHaveLength(0);
    });

    it("clears the window when its tag is closed", async () => {
      const { key } = await renderData([rolling(FORTY_FIVE_MIN)]);
      await screen.findByText(label(FORTY_FIVE_MIN));
      fireEvent.click(findTagCloseButton(label(FORTY_FIVE_MIN)));
      await expectRanges(key, []);
    });
  });

  describe("favorites", () => {
    it("adds a favorite the session owns as a static window", async () => {
      const favorite: Session.Range.StaticState = {
        variant: "static",
        key: "3a7b9c1d-5e2f-4a6b-8c0d-2e4f6a8b0c1d",
        name: "Hot fire",
        timeRange: { start: 1_000_000_000, end: 3_000_000_000 },
      };
      const { key } = await renderData([], [favorite]);
      openSearch();
      fireEvent.click(await screen.findByText("Hot fire"));
      await expectRanges(key, [
        {
          variant: "window",
          window: staticWindow(new TimeRange(1_000_000_000, 3_000_000_000)),
        },
      ]);
    });

    it("adds a favorite the Core holds as a key reference", async () => {
      const created = await createTestRange(client);
      const { key } = await renderData([], Session.Range.fromClient(created.payload));
      openSearch();
      fireEvent.click(await screen.findByText(created.name));
      await expectRanges(key, [{ variant: "key", key: created.key }]);
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

  describe("static window", () => {
    it("edits the start from its tag", async () => {
      const { key } = await renderData([fixed(14, 16)]);
      editTime("Start", "2026-08-23 15:00");
      await expectRanges(key, [fixed(15, 16)]);
    });

    it("edits the end from its tag", async () => {
      const { key } = await renderData([fixed(14, 16)]);
      editTime("End", "2026-08-23 17:00");
      await expectRanges(key, [fixed(14, 17)]);
    });

    it("rejects a bound that would cross the other", async () => {
      const { key } = await renderData([fixed(14, 16)]);
      editTime("Start", "2026-08-23 17:00");
      editTime("End", "2026-08-23 18:00");
      await expectRanges(key, [fixed(14, 18)]);
      editTime("End", "2026-08-23 13:00");
      editTime("Start", "2026-08-23 15:00");
      await expectRanges(key, [fixed(15, 18)]);
    });
  });

  it("removes a static window when its tag is closed", async () => {
    const { key } = await renderData([
      {
        variant: "window",
        window: staticWindow(new TimeRange(1_000_000_000, 2_000_000_000)),
      },
    ]);
    fireEvent.click(await screen.findByRole("button", { name: "Remove" }));
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
});
