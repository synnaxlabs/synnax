// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, type Synnax as Client } from "@synnaxlabs/client";
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

const streamOf = (data: string): ReadableStream<Uint8Array> =>
  new Response(data).body as ReadableStream<Uint8Array>;

describe("Export.fetchFileData", () => {
  it("streams the envelope and promotes its name", async () => {
    const body = JSON.stringify({ version: 1, type: "lineplot", name: "My Plot" });
    const exportFn = vi.fn().mockResolvedValue(streamOf(body));
    const client = { imex: { export: exportFn } } as unknown as Client;
    const id = lineplot.ontologyID("plot-1");
    const file = await Export.fetchFileData(client, id);
    expect(file).toEqual({ data: body, name: "My Plot" });
    expect(exportFn).toHaveBeenCalledWith(id, { encoding: "JSON" });
  });
});

describe("Export.use", () => {
  it("surfaces an error status and saves nothing when disconnected", async () => {
    const { result } = await renderHookWithConsole(() => ({
      run: Export.use(),
      notifications: Status.useNotifications(),
    }));
    act(() => result.current.run(lineplot.ontologyID("plot-1")));
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (s) => s.variant === "error" && s.message === "Failed to export lineplot",
        ),
      ).toBe(true),
    );
    expect(downloads.anchors).toHaveLength(0);
  });
});
