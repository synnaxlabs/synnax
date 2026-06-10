// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Layout } from "@/layout";
import { Log } from "@/log";
import { renderLog } from "@/log/toolbar/testutil";
import { Toolbar } from "@/log/toolbar/Toolbar";
import { type ConsolePreloadedState, renderWithConsole } from "@/testUtils";

const layoutState = (key: string, name: string): Layout.State => ({
  key,
  windowKey: MAIN_WINDOW,
  type: "log",
  name,
  location: "mosaic",
});

const preloadedState = (
  key: string,
  name: string,
  logState: Partial<Log.State> = {},
): ConsolePreloadedState => ({
  [Layout.SLICE_NAME]: {
    ...Layout.ZERO_SLICE_STATE,
    layouts: { ...Layout.ZERO_SLICE_STATE.layouts, [key]: layoutState(key, name) },
  },
  [Log.SLICE_NAME]: {
    ...Log.ZERO_SLICE_STATE,
    logs: { [key]: { ...Log.ZERO_STATE, key, ...logState } },
  },
});

const renderToolbar = (name = "Test Log") =>
  renderLog(Toolbar, { preloadedState: (key) => preloadedState(key, name) });

describe("log/toolbar/Toolbar", () => {
  describe("gating", () => {
    it("renders null when the log does not exist in state", () => {
      const { container } = renderWithConsole(<Toolbar layoutKey="no-log" />, {
        preloadedState: { [Log.SLICE_NAME]: Log.ZERO_SLICE_STATE },
      });
      expect(container.firstChild).toBeNull();
    });

    it("renders null when an upload is pending", () => {
      const key = uuid.create();
      const { container } = renderWithConsole(<Toolbar layoutKey={key} />, {
        preloadedState: preloadedState(key, "Pending", {
          pendingUpload: log.newZ.omit({ name: true }).parse({ key }),
        }),
      });
      expect(container.firstChild).toBeNull();
    });
  });

  describe("content", () => {
    it("displays the layout name in the title", async () => {
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
});
