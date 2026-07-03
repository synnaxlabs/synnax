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
  createTestClient,
  DataType,
  type lineplot,
  type Synnax,
} from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { LinePlot } from "@/platform/lineplot";

const client: Synnax = createTestClient();

describe("LinePlot.addChannelsToActivePlot", () => {
  let ch1: channel.Channel;
  let ch2: channel.Channel;

  const name = (prefix: string): string => `${prefix}_${id.create().replace(/-/g, "")}`;

  beforeAll(async () => {
    const idx = await client.channels.create({
      name: name("idx"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    ch1 = await client.channels.create({
      name: name("ch1"),
      dataType: DataType.FLOAT32,
      index: idx.key,
    });
    ch2 = await client.channels.create({
      name: name("ch2"),
      dataType: DataType.FLOAT32,
      index: idx.key,
    });
  });

  const createPlot = async (): Promise<lineplot.Key> => {
    const proj = await client.projects.create({
      name: `proj-${id.create()}`,
      layout: {},
    });
    const plot = await client.lineplots.create(proj.key, { name: "Plot" });
    return plot.key;
  };

  it("adds channels that are not yet on the y1 axis", async () => {
    const key = await createPlot();
    await LinePlot.addChannelsToActivePlot(client, key, [ch1.key, ch2.key]);
    await waitFor(async () => {
      const { channels } = await client.lineplots.retrieve({ key });
      expect(channels.y1).toContain(ch1.key);
      expect(channels.y1).toContain(ch2.key);
    });
  });

  it("does not dispatch when every channel is already present", async () => {
    const key = await createPlot();
    await LinePlot.addChannelsToActivePlot(client, key, [ch1.key]);
    await waitFor(async () => {
      const { channels } = await client.lineplots.retrieve({ key });
      expect(channels.y1).toContain(ch1.key);
    });

    const dispatchSpy = vi.spyOn(client.lineplots, "dispatch");
    await LinePlot.addChannelsToActivePlot(client, key, [ch1.key]);
    expect(dispatchSpy).not.toHaveBeenCalled();
    dispatchSpy.mockRestore();
  });
});
