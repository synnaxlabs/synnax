// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  DataType,
  lineplot,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { dynamicWindow, id, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  client,
  createPreloadedState,
  renderLinePlot,
} from "@/feature/lineplot/testutil";
import { Toolbar } from "@/feature/lineplot/toolbar";
import { findButton, getSwitch } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import {
  captureBrowserDownloads,
  getIconButton,
  readBlob,
  removeSaveFilePicker,
  uniqueName,
} from "@/testutil";

const roles = new RoleClients(client);

const renderToolbar = async (name = uniqueName("plot"), as?: Client) => ({
  name,
  ...(await renderLinePlot(Toolbar, {
    linePlot: { name },
    preloadedState: (key) => createPreloadedState(key),
    as,
  })),
});

describe("lineplot/toolbar/Toolbar", () => {
  it("switches tabs and records the active tab in the session store", async () => {
    const { key, store } = await renderToolbar();
    fireEvent.click(await screen.findByText("Axes"));
    expect(await screen.findByText("Lower bound")).toBeDefined();
    expect(
      documentIn(store.getState()[Session.LinePlot.SLICE_NAME], key)?.toolbar.activeTab,
    ).toBe("axes");
    fireEvent.click(screen.getByText("Data"));
    expect(await screen.findByText("Y2")).toBeDefined();
    expect(
      documentIn(store.getState()[Session.LinePlot.SLICE_NAME], key)?.toolbar.activeTab,
    ).toBe("data");
  });

  it("shows the lines empty state when no lines are plotted", async () => {
    await renderToolbar();
    fireEvent.click(await screen.findByText("Lines"));
    expect(await screen.findByText(/No lines plotted/)).toBeDefined();
  });

  it("opens the download CSV modal for the plot", async () => {
    const { name, result } = await renderToolbar();
    await screen.findByText("Data");
    fireEvent.click(getIconButton(result.container, "csv"));
    expect(await screen.findByText(`Download data for ${name} as CSV`)).toBeDefined();
  });

  describe("download CSV", () => {
    afterEach(() => {
      vi.restoreAllMocks();
    });

    // Creates a channel whose five samples stopped an hour ago, outside a 30 s window
    // that ends at the clock. Each sample sits 1 ns past a multiple of 256 ns, the
    // float step near now, so an end 1 ns past the last one rounds below it as a float.
    const createStopped = async (suffix: string): Promise<channel.Channel> => {
      const index = await client.channels.create({
        name: `toolbar_csv_time_${suffix}`,
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data = await client.channels.create({
        name: `toolbar_csv_data_${suffix}`,
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      const hourAgo = TimeStamp.now().sub(TimeSpan.hours(1)).valueOf();
      const start = new TimeStamp(hourAgo - (hourAgo % 256n) + 1n);
      await client.write(start, {
        [index.key]: Array.from({ length: 5 }, (_, i) =>
          start.add(TimeSpan.seconds(i)),
        ),
        [data.key]: [1, 2, 3, 4, 5],
      });
      return data;
    };

    // Opens the CSV download of a live plot of the channels and returns its Download
    // button once it is enabled.
    const openLive = async (channels: channel.Key[]): Promise<HTMLElement> => {
      const key = uuid.create();
      const lines = channels.map((yChannel) => {
        const line = { yAxis: "y1" as const, yChannel, xChannel: 0 };
        return { key: lineplot.lineKey(key, line), ...line };
      });
      const { result } = await renderLinePlot(Toolbar, {
        linePlot: {
          key,
          name: uniqueName("plot"),
          channels: { y1: channels },
          lines,
          ranges: [{ variant: "window", window: dynamicWindow(TimeSpan.seconds(30)) }],
        },
        preloadedState: (key) => createPreloadedState(key),
      });
      await screen.findByText("Data");
      fireEvent.click(getIconButton(result.container, "csv"));
      return await waitFor(() => {
        const button = findButton("Download");
        expect(button.className).not.toContain("lyra--disabled");
        return button;
      });
    };

    it("should end a live window at the newest sample of the plot's channels", async () => {
      removeSaveFilePicker();
      const downloads = captureBrowserDownloads();
      const data = await createStopped(id.create().replace(/-/g, "_"));
      const download = await openLive([data.key]);
      await act(async () => {
        fireEvent.click(download);
      });
      await waitFor(() => expect(downloads.blobs).toHaveLength(1));
      const rows = (await readBlob(downloads.blobs[0])).trim().split("\n").slice(1);
      expect(rows.map((r) => Number(r.split(",").at(-1)))).toEqual([1, 2, 3, 4, 5]);
    });

    it("should open the download of a plot with a calculation over virtual channels", async () => {
      const suffix = id.create().replace(/-/g, "_");
      const data = await createStopped(suffix);
      const source = await client.channels.create({
        name: `toolbar_csv_virtual_${suffix}`,
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const calculated = await client.channels.create({
        name: `toolbar_csv_calculated_${suffix}`,
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: `return ${source.name} * 2`,
      });
      const download = await openLive([data.key, calculated.key]);
      expect(download.className).not.toContain("lyra--disabled");
    });
  });

  it("renames the plot from the properties tab", async () => {
    const { key, name } = await renderToolbar();
    fireEvent.click(await screen.findByText("Properties"));
    await screen.findByText("Show title");
    const newName = uniqueName("renamed");
    const input = await waitFor(() => screen.getByDisplayValue(name));
    fireEvent.change(input, { target: { value: newName } });
    // The title commits on blur, so a keystroke alone must not reach the server.
    await act(async () => {});
    expect((await client.lineplots.retrieve(key)).name).toBe(name);
    fireEvent.blur(input);
    expect(await screen.findByText(newName)).toBeDefined();
    await waitFor(async () => {
      const remote = await client.lineplots.retrieve(key);
      expect(remote.name).toBe(newName);
    });
  });

  it("toggles title visibility from the properties tab", async () => {
    const { key } = await renderToolbar();
    fireEvent.click(await screen.findByText("Properties"));
    await screen.findByText("Show title");
    const titleSwitch = getSwitch("Show title");
    expect(titleSwitch.checked).toBe(false);
    fireEvent.click(titleSwitch);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.title.visible).toBe(true);
    });
    expect(titleSwitch.checked).toBe(true);
  });

  it("toggles legend visibility from the properties tab", async () => {
    const { key } = await renderToolbar();
    fireEvent.click(await screen.findByText("Properties"));
    await screen.findByText("Show legend");
    const legendSwitch = getSwitch("Show legend");
    expect(legendSwitch.checked).toBe(true);
    fireEvent.click(legendSwitch);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.legend.hidden).toBe(true);
    });
  });
});

describe("lineplot/toolbar/Toolbar permissions", () => {
  it("should withhold the editing tabs from a viewer", async () => {
    const { name } = await renderToolbar(undefined, await roles.get("Viewer"));
    expect(await screen.findByText(name)).toBeTruthy();
    expect(screen.queryByText("Axes")).toBeNull();
    expect(screen.queryByText("Lines")).toBeNull();
    expect(screen.queryByText("Data")).toBeNull();
  });

  it("should withhold the editing controls from a viewer", async () => {
    const { name } = await renderToolbar(undefined, await roles.get("Viewer"));
    expect(await screen.findByText(`${name} is not editable`)).toBeTruthy();
    expect(screen.queryByText("Y2")).toBeNull();
  });
});
