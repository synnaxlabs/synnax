// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";

import { Layout } from "@/layout";
import { moveMosaicTab } from "@/layout/slice";
import { LinePlot } from "@/lineplot";
import { registerMacro } from "@/perf/macros/registry";
import { type MacroContext, type MacroStep } from "@/perf/macros/types";

const PERF_CHANNELS = [
  "sy_node_1_metrics_mem_percentage",
  "sy_node_1_metrics_cpu_percentage",
];

export const linePlotMacro: MacroStep[] = [
  {
    name: "Create Line Plot",
    execute: async (ctx: MacroContext) => {
      const { key } = ctx.placer(
        LinePlot.create({ name: `Perf Test Plot ${Date.now()}`, location: "mosaic" }),
      );
      ctx.createdLayoutKeys.push(key);
    },
  },
  {
    name: "Snap to Right",
    execute: async (ctx: MacroContext) => {
      const plotKey = ctx.createdLayoutKeys[ctx.createdLayoutKeys.length - 1];
      if (plotKey == null) return;
      ctx.dispatch(
        moveMosaicTab({ windowKey: MAIN_WINDOW, key: 1, tabKey: plotKey, loc: "right" }),
      );
    },
  },
  {
    name: "Add Channels",
    execute: async (ctx: MacroContext) => {
      const plotKey = ctx.createdLayoutKeys[ctx.createdLayoutKeys.length - 1];
      if (plotKey == null || ctx.client == null) return;

      const channels = await ctx.client.channels.retrieve(PERF_CHANNELS);
      if (channels.length === 0) return;

      const indexChannel = channels.find((ch) => ch.isIndex);
      if (indexChannel != null)
        ctx.dispatch(
          LinePlot.setXChannel({ key: plotKey, axisKey: "x1", channel: indexChannel.key }),
        );

      ctx.dispatch(
        LinePlot.setYChannels({
          key: plotKey,
          axisKey: "y1",
          channels: channels.filter((ch) => !ch.isIndex).map((ch) => ch.key),
          mode: "set",
        }),
      );
    },
  },
  {
    name: "Close Plot",
    execute: async (ctx: MacroContext) => {
      const plotKey = ctx.createdLayoutKeys.pop();
      if (plotKey == null) return;
      ctx.dispatch(Layout.remove({ keys: [plotKey] }));
    },
  },
];

registerMacro({
  type: "linePlot",
  name: "Line Plot",
  description: "Creates a line plot, snaps to right, adds channels, then closes",
  category: "lineplot",
  factory: () => linePlotMacro,
});
