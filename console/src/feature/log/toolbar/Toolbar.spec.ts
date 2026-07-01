// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Toolbar } from "@/feature/log/toolbar/Toolbar";
import { renderLog } from "@/primitive/log/toolbar/testutil";
import { Session } from "@/session";
import { type ConsolePreloadedState } from "@/testutil/testutil";

const layoutState = (key: string, name: string): Session.Layout.State => ({
  key,
  windowKey: MAIN_WINDOW,
  type: "log",
  name,
  location: "mosaic",
});

const preloadedState = (
  key: string,
  name: string,
  logState: Partial<Session.Log.State> = {},
): ConsolePreloadedState => ({
  [Session.Layout.SLICE_NAME]: {
    ...Session.Layout.ZERO_SLICE_STATE,
    layouts: {
      ...Session.Layout.ZERO_SLICE_STATE.layouts,
      [key]: layoutState(key, name),
    },
  },
  [Session.Log.SLICE_NAME]: {
    ...Session.Log.ZERO_SLICE_STATE,
    logs: { [key]: { ...Session.Log.ZERO_STATE, ...logState } },
  },
});

const renderToolbar = (name = "Test Log") =>
  renderLog(Toolbar, {
    log: { name },
    preloadedState: (key) => preloadedState(key, name),
  });

describe("log/toolbar/Toolbar", () => {
  it("displays the log name in the title", async () => {
    await renderToolbar("My Log");
    expect(await screen.findByText("My Log")).toBeDefined();
  });

  it("renders both tab buttons", async () => {
    await renderToolbar();
    expect(await screen.findByText("Channels")).toBeDefined();
    expect(screen.getByText("Properties")).toBeDefined();
  });

  it("defaults to the channels tab", async () => {
    await renderToolbar();
    expect(await screen.findByText("Add a channel...")).toBeDefined();
  });

  it("switches to the Properties tab when clicked", async () => {
    await renderToolbar();
    await screen.findByText("Add a channel...");
    fireEvent.click(screen.getByText("Properties"));
    expect(await screen.findByText("Show Channel Names")).toBeDefined();
  });

  it("switches back to the channels tab", async () => {
    await renderToolbar();
    await screen.findByText("Add a channel...");
    fireEvent.click(screen.getByText("Properties"));
    await screen.findByText("Show Channel Names");
    fireEvent.click(screen.getByText("Channels"));
    expect(await screen.findByText("Add a channel...")).toBeDefined();
  });
});
