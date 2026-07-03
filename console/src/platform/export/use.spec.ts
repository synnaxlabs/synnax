// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { act, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Export } from "@/platform/export";
import {
  captureBrowserDownloads,
  type CapturedDownloads,
  renderHookWithConsole,
} from "@/testutil";

let downloads: CapturedDownloads;

beforeEach(() => {
  downloads = captureBrowserDownloads();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("Export.use", () => {
  it("extracts with the key and current store/client, then saves the result", async () => {
    const extract = vi
      .fn()
      .mockResolvedValue({ name: "My Plot", data: '{"key":"plot-1"}' });
    const { result, store } = await renderHookWithConsole(() =>
      Export.use(extract, "lineplot"),
    );
    act(() => result.current("plot-1"));
    await waitFor(() => expect(extract).toHaveBeenCalledTimes(1));
    expect(extract).toHaveBeenCalledWith("plot-1", { store, client: null });
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toEqual("My Plot.json");
  });

  it("surfaces an error status and saves nothing when the extractor fails", async () => {
    const extract = vi.fn().mockRejectedValue(new Error("cannot extract"));
    const { result } = await renderHookWithConsole(() => ({
      run: Export.use(extract, "lineplot"),
      notifications: Status.useNotifications(),
    }));
    act(() => result.current.run("plot-1"));
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (s) => s.variant === "error" && s.message === "Failed to export lineplot",
        ),
      ).toBe(true),
    );
    expect(extract).toHaveBeenCalledTimes(1);
    expect(downloads.anchors).toHaveLength(0);
  });
});
