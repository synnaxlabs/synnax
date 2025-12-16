// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { type WorkflowContext, type WorkflowStep } from "@/perf/workflows/types";

/**
 * Workflow to create a new line plot.
 */
export const createLinePlotWorkflow = (): WorkflowStep[] => [
  {
    name: "Create Line Plot",
    execute: async (ctx: WorkflowContext) => {
      const timestamp = Date.now();
      const { key } = ctx.placer(
        LinePlot.create({
          name: `Perf Test Plot ${timestamp}`,
          location: "mosaic",
        }),
      );
      ctx.createdLayoutKeys.push(key);
    },
    delayAfterMs: 500,
  },
];

/**
 * Workflow to add channels to an existing line plot.
 */
export const addChannelsToPlotWorkflow = (): WorkflowStep[] => [
  {
    name: "Add Channels to Plot",
    execute: async (ctx: WorkflowContext) => {
      // Find the most recently created plot
      const plotKey = ctx.createdLayoutKeys[ctx.createdLayoutKeys.length - 1];
      if (plotKey == null) 
        throw new Error("No plot available to add channels to");
      

      // Check if we have channels available
      if (ctx.availableChannelKeys.length === 0 && ctx.client != null) {
        // Try to fetch some channels
        const channels = await ctx.client.channels.retrieve({ limit: 10 });
        ctx.availableChannelKeys = channels.map((ch) => ch.key);
      }

      if (ctx.availableChannelKeys.length === 0) 
        // No channels available, skip
        return;
      

      // Find an index channel and data channels
      const channelKeys = ctx.availableChannelKeys;

      // Use first channel as X axis (typically a timestamp channel)
      const xChannel = channelKeys[0];
      ctx.dispatch(
        LinePlot.setXChannel({
          key: plotKey,
          axisKey: "x1",
          channel: xChannel,
        }),
      );

      // Add remaining channels as Y axis (up to 3)
      const yChannels = channelKeys.slice(1, 4);
      if (yChannels.length > 0) 
        ctx.dispatch(
          LinePlot.setYChannels({
            key: plotKey,
            axisKey: "y1",
            channels: yChannels,
            mode: "set",
          }),
        );
      
    },
    delayAfterMs: 1000,
  },
];

/**
 * Workflow to simulate pan and zoom operations on a plot.
 */
export const panZoomPlotWorkflow = (): WorkflowStep[] => [
  {
    name: "Pan Plot",
    execute: async (ctx: WorkflowContext) => {
      const plotKey = ctx.createdLayoutKeys[ctx.createdLayoutKeys.length - 1];
      if (plotKey == null) return;

      // Simulate panning by updating viewport
      ctx.dispatch(
        LinePlot.storeViewport({
          key: plotKey,
          pan: { x: Math.random() * 100, y: Math.random() * 50 },
          zoom: { width: 1000, height: 500 },
        }),
      );
    },
    delayAfterMs: 500,
  },
  {
    name: "Zoom Plot",
    execute: async (ctx: WorkflowContext) => {
      const plotKey = ctx.createdLayoutKeys[ctx.createdLayoutKeys.length - 1];
      if (plotKey == null) return;

      // Simulate zooming
      ctx.dispatch(
        LinePlot.storeViewport({
          key: plotKey,
          pan: { x: 50, y: 25 },
          zoom: { width: 500 + Math.random() * 500, height: 250 + Math.random() * 250 },
        }),
      );
    },
    delayAfterMs: 500,
  },
];

/**
 * Workflow to close the most recently created plot.
 */
export const closePlotWorkflow = (): WorkflowStep[] => [
  {
    name: "Close Plot",
    execute: async (ctx: WorkflowContext) => {
      const plotKey = ctx.createdLayoutKeys.pop();
      if (plotKey == null) return;

      ctx.dispatch(Layout.remove({ keys: [plotKey] }));
    },
    delayAfterMs: 500,
  },
];
