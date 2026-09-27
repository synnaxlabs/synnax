// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { TimeSpan } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { client, renderLinePlot } from "@/feature/lineplot/testutil";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { assertDefined, findTagCloseButton } from "@/testutil";

const renderData = async (
  ranges?: lineplot.New["ranges"],
  favorites: Session.Range.State[] = [],
) => {
  const handle = await renderLinePlot(LinePlot.Toolbar, {
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

const openSearch = (): HTMLElement => {
  fireEvent.click(screen.getByRole("button", { name: "Ranges" }));
  return screen.getByPlaceholderText("Search favorites or type a duration");
};

const search = (term: string): void => {
  fireEvent.change(openSearch(), { target: { value: term } });
};

// The rolling window's tag shows its span too, so a preset is told apart by its toggle.
const clickPreset = (span: string): void => {
  const preset = screen
    .getAllByRole("button", { name: span })
    .find((b) => b.hasAttribute("aria-pressed"));
  assertDefined(preset);
  fireEvent.click(preset);
};

const findOption = (text: string): HTMLElement | undefined =>
  screen.queryAllByRole("option").find((o) => o.textContent?.includes(text));

const retrieveX1 = async (key: string): Promise<lineplot.XAxisRanges> =>
  (await client.lineplots.retrieve(key)).ranges.x1;

const expectRolling = async (key: string, span?: TimeSpan): Promise<void> =>
  await waitFor(async () =>
    expect((await retrieveX1(key)).rolling).toEqual(
      span == null ? undefined : Number(span),
    ),
  );

const ROLLING_45M: lineplot.New["ranges"] = {
  x1: { rolling: Number(TimeSpan.minutes(45)) },
};

const STATIC: lineplot.StaticRange = {
  variant: "static",
  key: "0e4f6a1b-2c3d-4e5f-8a9b-1c2d3e4f5a6b",
  name: "Burn",
  start: "1000000000",
  end: "2000000000",
};

describe("lineplot/toolbar/Data", () => {
  describe("rolling window", () => {
    it("lists the presets as a row, apart from the favorites", async () => {
      await renderData();
      openSearch();
      expect(screen.getByRole("button", { name: "5m" })).toBeDefined();
      expect(screen.queryAllByRole("option")).toHaveLength(0);
    });

    it("keeps one window, replacing it with the next preset picked", async () => {
      const { key } = await renderData();
      openSearch();
      clickPreset("5m");
      await expectRolling(key, TimeSpan.minutes(5));
      clickPreset("1h");
      await expectRolling(key, TimeSpan.hours(1));
    });

    it("clears the window when its preset is clicked again", async () => {
      const { key } = await renderData({
        x1: { rolling: Number(TimeSpan.minutes(5)) },
      });
      openSearch();
      clickPreset("5m");
      await expectRolling(key, undefined);
    });

    it("offers a typed window when the search is a duration", async () => {
      await renderData();
      search("45m");
      expect(findOption("45m")).toBeDefined();
    });

    it("offers no typed window when the search is not a duration", async () => {
      await renderData();
      const input = openSearch();
      fireEvent.change(input, { target: { value: "banana" } });
      expect(screen.queryAllByRole("option")).toHaveLength(0);
      fireEvent.change(input, { target: { value: "0s" } });
      expect(screen.queryAllByRole("option")).toHaveLength(0);
    });

    it("sets the window to the typed span", async () => {
      const { key } = await renderData({
        x1: { rolling: Number(TimeSpan.minutes(5)) },
      });
      search("1h 30m");
      const option = findOption("1h 30m");
      assertDefined(option);
      fireEvent.click(option);
      await expectRolling(key, TimeSpan.minutes(90));
    });

    it("sets the window to the typed span with Enter", async () => {
      const { key } = await renderData();
      search("2w");
      act(() => {
        fireEvent.keyDown(window, { code: "Enter" });
      });
      await expectRolling(key, TimeSpan.days(14));
    });

    it("clears the window from the typed option that matches it", async () => {
      const { key } = await renderData(ROLLING_45M);
      search("45m");
      const option = findOption("45m");
      assertDefined(option);
      expect(option.getAttribute("aria-selected")).toBe("true");
      fireEvent.click(option);
      await expectRolling(key, undefined);
    });

    it("edits the window from its tag", async () => {
      const { key } = await renderData(ROLLING_45M);
      fireEvent.click(await screen.findByText("45m"));
      const field = screen.getByRole("textbox");
      fireEvent.change(field, { target: { value: "10m" } });
      fireEvent.keyDown(field, { key: "Enter" });
      await expectRolling(key, TimeSpan.minutes(10));
    });

    it("clears the window when its tag is closed", async () => {
      const { key } = await renderData(ROLLING_45M);
      await screen.findByText("45m");
      fireEvent.click(findTagCloseButton("45m"));
      await expectRolling(key, undefined);
    });
  });

  describe("favorites", () => {
    it("adds a favorite the session owns as a static range", async () => {
      const favorite: Session.Range.StaticState = {
        variant: "static",
        key: "3a7b9c1d-5e2f-4a6b-8c0d-2e4f6a8b0c1d",
        name: "Hot fire",
        timeRange: { start: 1_000_000_000, end: 3_000_000_000 },
      };
      const { key } = await renderData(undefined, [favorite]);
      openSearch();
      fireEvent.click(await screen.findByText("Hot fire"));
      await waitFor(async () => {
        const [range] = (await retrieveX1(key)).ranges;
        expect(range).toMatchObject({
          variant: "static",
          name: "Hot fire",
          start: "1000000000",
          end: "3000000000",
        });
      });
    });

    it("adds a favorite the Core holds as a persisted range", async () => {
      const created = await createTestRange(client);
      const { key } = await renderData(
        undefined,
        Session.Range.fromClient(created.payload),
      );
      openSearch();
      fireEvent.click(await screen.findByText(created.name));
      await waitFor(async () =>
        expect((await retrieveX1(key)).ranges).toEqual([
          { variant: "persisted", key: created.key },
        ]),
      );
    });

    it("names a persisted range's tag after the Core range", async () => {
      const created = await createTestRange(client);
      await renderData({
        x1: { ranges: [{ variant: "persisted", key: created.key }] },
      });
      expect(await screen.findByText(created.name)).toBeDefined();
    });
  });

  it("removes a static range when its tag is closed", async () => {
    const { key, result } = await renderData({ x1: { ranges: [STATIC] } });
    const close = await waitFor(() => {
      const btn = result.container.querySelector<HTMLElement>(
        ".console-range-select__editable .pluto-tag__close",
      );
      assertDefined(btn);
      return btn;
    });
    fireEvent.click(close);
    await waitFor(async () => expect((await retrieveX1(key)).ranges).toEqual([]));
  });
});
