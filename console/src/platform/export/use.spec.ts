// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { act, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Export } from "@/platform/export";
import {
  captureBrowserDownloads,
  type CapturedDownloads,
  renderHookWithConsole,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

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
    const stream = vi.fn();
    act(() => result.current.run({ stream, name: "My Plot", extension: "json" }));
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (s) =>
            s.variant === "error" &&
            s.name === "My Plot" &&
            s.message === "Failed to export",
        ),
      ).toBe(true),
    );
    // The disconnected check runs before the stream fetcher, so no request is made.
    expect(stream).not.toHaveBeenCalled();
    expect(downloads.anchors).toHaveLength(0);
  });
});

describe("Export.useResource", () => {
  it("saves the export to a file named after the resource", async () => {
    const proj = await client.projects.create({ name: uniqueName("project") });
    const plot = await client.lineplots.create(proj.key, { name: uniqueName("plot") });
    const { result } = await renderHookWithConsole(() => Export.useResource(), {
      client,
    });
    act(() => result.current(lineplot.resourceID(plot.key)));
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toEqual(`${plot.name}.json`);
  });

  it("surfaces an error status and saves nothing when disconnected", async () => {
    const { result } = await renderHookWithConsole(() => ({
      run: Export.useResource(),
      notifications: Status.useNotifications(),
    }));
    act(() => result.current.run(lineplot.resourceID("plot-1")));
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (s) => s.variant === "error" && s.message === "Failed to export resource",
        ),
      ).toBe(true),
    );
    expect(downloads.anchors).toHaveLength(0);
  });
});
