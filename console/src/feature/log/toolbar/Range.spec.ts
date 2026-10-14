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
import { TimeSpan } from "@synnaxlabs/x";
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
  return screen.getByPlaceholderText("Search ranges or type a duration");
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

  it("should set a rolling window from a typed duration", async () => {
    const { key } = await render();
    fireEvent.change(await openSearch(), { target: { value: "1h 30m" } });
    const option = findOption("1h 30m");
    assertDefined(option);
    fireEvent.click(option);
    await expectRange(key, rolling(TimeSpan.minutes(90)));
  });

  it("should set a typed duration with Enter", async () => {
    const { key } = await render();
    fireEvent.change(await openSearch(), { target: { value: "2w" } });
    act(() => {
      fireEvent.keyDown(window, { code: "Enter" });
    });
    await expectRange(key, rolling(TimeSpan.days(14)));
  });

  it("should offer no typed window when the search is not a duration", async () => {
    await render();
    fireEvent.change(await openSearch(), { target: { value: "banana" } });
    expect(findOption("banana")).toBeUndefined();
  });

  it("should edit the rolling window from its tag", async () => {
    const { key } = await render(rolling(TimeSpan.minutes(45)));
    fireEvent.click(await screen.findByText("45m"));
    const field = screen.getByRole("textbox");
    fireEvent.change(field, { target: { value: "10m" } });
    fireEvent.keyDown(field, { key: "Enter" });
    await expectRange(key, rolling(TimeSpan.minutes(10)));
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
