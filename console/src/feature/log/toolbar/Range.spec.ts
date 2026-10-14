// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/log/toolbar/Range";
import {
  client,
  renderLog,
  type RenderLogOptions,
} from "@/feature/log/toolbar/testutil";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { assertDefined } from "@/testutil";

const roles = new RoleClients(client);

const rolling = (span: TimeSpan): ranger.Ref => ({
  variant: "window",
  window: { variant: "dynamic", timeSpan: Number(span) },
});

const STATIC: ranger.Ref = {
  variant: "window",
  window: {
    variant: "static",
    timeRange: { start: "1000000000", end: "3000000000" },
  },
};

const render = async (
  range?: ranger.Ref,
  favorites: Session.Range.State[] = [],
  options: Omit<RenderLogOptions, "log" | "preloadedState"> = {},
) =>
  await renderLog(Range, {
    ...options,
    log: range == null ? {} : { range },
    preloadedState: () => ({
      [Session.Range.SLICE_NAME]: {
        ...Session.Range.ZERO_SLICE_STATE,
        ranges: favorites,
      },
    }),
  });

const trigger = async (): Promise<HTMLElement> =>
  await screen.findByRole("button", { name: "Range" });

const openSearch = async (): Promise<HTMLElement> => {
  fireEvent.click(await trigger());
  return screen.getByPlaceholderText("Search ranges or type a span");
};

// The rolling window's tag shows its span too, so a preset is told apart by its toggle.
const clickPreset = (span: string): void => {
  const preset = screen
    .getAllByRole("button", { name: span })
    .find((b) => b.hasAttribute("aria-pressed"));
  assertDefined(preset);
  fireEvent.click(preset);
};

const expectRange = async (key: string, range: ranger.Ref): Promise<void> =>
  await waitFor(async () =>
    expect((await client.logs.retrieve(key)).range).toEqual(range),
  );

describe("log/toolbar/Range", () => {
  it("should show the rolling 7 day window for a new log", async () => {
    await render();
    expect((await trigger()).textContent).toContain("7d");
  });

  it("should set a rolling window from a preset", async () => {
    const { key } = await render();
    await openSearch();
    clickPreset("1h");
    await expectRange(key, rolling(TimeSpan.hours(1)));
  });

  const typeSpan = (text: string, key: string = "Enter"): void => {
    const field = screen.getByRole<HTMLInputElement>("textbox", {
      name: "Rolling span",
    });
    act(() => field.focus());
    fireEvent.change(field, { target: { value: text } });
    act(() => {
      fireEvent.keyDown(field, { key, code: key });
    });
  };

  it("should set a rolling window from a typed span", async () => {
    const { key } = await render();
    await openSearch();
    typeSpan("1h 30m");
    await expectRange(key, rolling(TimeSpan.minutes(90)));
  });

  it("should let a preset click win over an edited span field", async () => {
    const { key } = await render();
    await openSearch();
    const field = screen.getByRole("textbox", { name: "Rolling span" });
    act(() => field.focus());
    fireEvent.change(field, { target: { value: "2h" } });
    act(() => field.blur());
    clickPreset("1h");
    await expectRange(key, rolling(TimeSpan.hours(1)));
  });

  it("should set a rolling window from a span typed in the search", async () => {
    const { key } = await render();
    const search = await openSearch();
    fireEvent.change(search, { target: { value: "14min 7s" } });
    act(() => {
      fireEvent.keyDown(search, { key: "Enter", code: "Enter" });
    });
    await expectRange(key, rolling(TimeSpan.seconds(14 * 60 + 7)));
  });

  it("should pick the hovered range on Enter when the search is not a span", async () => {
    const created = await createTestRange(client);
    const { key } = await render(undefined, Session.Range.fromClient(created.payload));
    const search = await openSearch();
    fireEvent.change(search, { target: { value: created.name } });
    await screen.findByText(created.name);
    act(() => {
      fireEvent.keyDown(search, { key: "Enter", code: "Enter" });
    });
    await expectRange(key, { variant: "key", key: created.key });
  });

  it("should ignore a typed span that does not parse", async () => {
    const { key } = await render(rolling(TimeSpan.hours(1)));
    await openSearch();
    typeSpan("banana");
    await expectRange(key, rolling(TimeSpan.hours(1)));
  });

  it("should show a span no preset matches in the span field", async () => {
    await render(rolling(TimeSpan.days(14)));
    await openSearch();
    expect(screen.getByRole("textbox", { name: "Rolling span" })).toHaveProperty(
      "value",
      "14d",
    );
  });

  it("should not list a typed duration as a search result", async () => {
    await render();
    fireEvent.change(await openSearch(), { target: { value: "14d" } });
    expect(screen.queryAllByRole("option")).toHaveLength(0);
  });

  it("should show a fixed window's bounds for editing", async () => {
    await render(STATIC);
    await openSearch();
    expect(screen.getByText("Fixed")).toBeDefined();
  });

  it("should keep an edited fixed bound when Enter commits it", async () => {
    const favorite: Session.Range.StaticState = {
      variant: "static",
      key: "3a7b9c1d-5e2f-4a6b-8c0d-2e4f6a8b0c1d",
      name: "Hot fire",
      timeRange: { start: 5_000_000_000, end: 6_000_000_000 },
    };
    const { key } = await render(STATIC, [favorite]);
    await openSearch();
    await screen.findByText("Hot fire");
    const [startBound] = document.querySelectorAll(".pluto-time-editor__trigger");
    fireEvent.click(startBound);
    const field = screen
      .getAllByRole<HTMLInputElement>("textbox")
      .find(
        (f) =>
          f.placeholder !== "Search ranges or type a span" &&
          f.ariaLabel !== "Rolling span",
      );
    assertDefined(field);
    const start = new TimeStamp(2_000_000_000n);
    fireEvent.change(field, { target: { value: start.toString("dateTime", "local") } });
    act(() => {
      fireEvent.keyDown(field, { key: "Enter", code: "Enter" });
    });
    await waitFor(async () => {
      const { range } = await client.logs.retrieve(key);
      expect(range).toMatchObject({ variant: "window", window: { variant: "static" } });
      expect(range).not.toEqual(STATIC);
    });
  });

  it("should store a range only the session holds as its fixed window", async () => {
    const favorite: Session.Range.StaticState = {
      variant: "static",
      key: "3a7b9c1d-5e2f-4a6b-8c0d-2e4f6a8b0c1d",
      name: "Hot fire",
      timeRange: { start: 1_000_000_000, end: 3_000_000_000 },
    };
    const { key } = await render(undefined, [favorite]);
    await openSearch();
    fireEvent.click(await screen.findByText("Hot fire"));
    await expectRange(key, STATIC);
  });

  it("should store a range the Core holds by key", async () => {
    const created = await createTestRange(client);
    const { key } = await render(undefined, Session.Range.fromClient(created.payload));
    await openSearch();
    fireEvent.click(await screen.findByText(created.name));
    await expectRange(key, { variant: "key", key: created.key });
  });

  it("should name a saved range the session does not hold", async () => {
    const created = await createTestRange(client);
    await render({ variant: "key", key: created.key });
    await waitFor(async () =>
      expect((await trigger()).textContent).toContain(created.name),
    );
  });

  it("should freeze the select for a viewer", async () => {
    await render(undefined, [], { as: await roles.get("Viewer") });
    const button = await trigger();
    await waitFor(() => expect(button.getAttribute("aria-disabled")).toBe("true"));
  });
});
