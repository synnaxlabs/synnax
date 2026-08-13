// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
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
  it("surfaces an error status and saves nothing when disconnected", async () => {
    const { result } = await renderHookWithConsole(() => ({
      run: Export.use(),
      notifications: Status.useNotifications(),
    }));
    act(() =>
      result.current.run({ id: lineplot.ontologyID("plot-1"), name: "My Plot" }),
    );
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (s) => s.variant === "error" && s.message === "Failed to export My Plot",
        ),
      ).toBe(true),
    );
    expect(downloads.anchors).toHaveLength(0);
  });
});
