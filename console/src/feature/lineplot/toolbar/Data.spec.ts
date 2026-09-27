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
import { Range } from "@/platform/range";
import { assertDefined, findTagCloseButton } from "@/testutil";

const renderData = async (ranges?: lineplot.New["ranges"]) => {
  const handle = await renderLinePlot(LinePlot.Toolbar, {
    linePlot: ranges === undefined ? {} : { ranges },
  });
  await screen.findByText("Ranges");
  return handle;
};

const openSearch = (): HTMLElement => {
  fireEvent.click(screen.getByRole("button", { name: "Ranges" }));
  return screen.getByPlaceholderText("Search ranges...");
};

const search = (term: string): void => {
  fireEvent.change(openSearch(), { target: { value: term } });
};

const queryCustomOption = (): HTMLElement | null =>
  screen.queryAllByRole("option").find((o) => o.textContent?.includes("Custom")) ??
  null;

const CUSTOM_45M: lineplot.New["ranges"] = {
  x1: [Range.CUSTOM_KEY],
  x2: [],
  custom: { variant: "dynamic", span: Number(TimeSpan.minutes(45)) },
};

describe("lineplot/toolbar/Data", () => {
  it("offers a custom window when the search is a duration", async () => {
    await renderData();
    search("45m");
    expect(queryCustomOption()?.textContent).toContain("45m");
  });

  it("offers no custom window when the search is not a duration", async () => {
    await renderData();
    const input = openSearch();
    fireEvent.change(input, { target: { value: "banana" } });
    expect(queryCustomOption()).toBeNull();
    fireEvent.change(input, { target: { value: "0s" } });
    expect(queryCustomOption()).toBeNull();
  });

  it("selects the custom window and persists its span", async () => {
    const { key } = await renderData();
    search("1h 30m");
    const option = queryCustomOption();
    assertDefined(option);
    fireEvent.click(option);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.ranges.x1).toEqual([Range.CUSTOM_KEY]);
      expect(plot.ranges.custom).toEqual({
        variant: "dynamic",
        span: Number(TimeSpan.minutes(90)),
      });
    });
  });

  it("selects the custom window with Enter", async () => {
    const { key } = await renderData();
    search("2w");
    act(() => {
      fireEvent.keyDown(window, { code: "Enter" });
    });
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.ranges.custom).toEqual({
        variant: "dynamic",
        span: Number(TimeSpan.days(14)),
      });
    });
  });

  it("shows the stored window on the custom tag", async () => {
    await renderData(CUSTOM_45M);
    const tag = (await screen.findByText("45m")).closest(".pluto-tag");
    expect(tag).not.toBeNull();
  });

  it("edits the stored window from the custom tag", async () => {
    const { key } = await renderData(CUSTOM_45M);
    fireEvent.click(await screen.findByText("45m"));
    const field = screen.getByRole("textbox");
    fireEvent.change(field, { target: { value: "10m" } });
    fireEvent.keyDown(field, { key: "Enter" });
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.ranges.custom).toEqual({
        variant: "dynamic",
        span: Number(TimeSpan.minutes(10)),
      });
    });
  });

  it("deselects the custom window when its tag is closed", async () => {
    const { key } = await renderData(CUSTOM_45M);
    await screen.findByText("45m");
    fireEvent.click(findTagCloseButton("45m"));
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.ranges.x1).toEqual([]);
    });
  });
});
