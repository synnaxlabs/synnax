// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DisconnectedError, type task } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { DataType, strings, unique } from "@synnaxlabs/x";

import { LinePlot } from "@/platform/lineplot";

/** @returns the channels of a read task's config that a plot of the task shows. */
export type PlotChannels = (config: unknown) => channel.Key[];

/** An entry of a read task config that fills one channel. */
export interface PlotEntry {
  channel: channel.Key;
  disabled: boolean;
}

/** @returns the channels of the enabled entries that the task has created. */
export const enabledChannels = (entries: PlotEntry[]): channel.Key[] =>
  entries
    .filter(({ channel, disabled }) => !disabled && channel !== 0)
    .map(({ channel }) => channel);

/** A read task to plot, with the function that finds the channels of its config. */
export interface PlotTask extends Pick<task.Task, "name" | "config"> {
  plotChannels: PlotChannels;
}

const isPlottable = ({ isIndex, dataType }: channel.Channel): boolean =>
  !isIndex && dataType.isNumeric && !dataType.equals(DataType.TIMESTAMP);

/** @returns a function that opens one line plot of the tasks' numeric channels. */
export const usePlot = (): ((tasks: PlotTask[]) => void) => {
  const client = Synnax.use();
  const create = LinePlot.useCreate();
  const handleError = Status.useErrorHandler();
  return (tasks) =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const keys = unique.unique(
        tasks.flatMap(({ config, plotChannels }) => plotChannels(config)),
      );
      const channels = keys.length === 0 ? [] : await client.channels.retrieve(keys);
      const y1 = channels.filter(isPlottable).map(({ key }) => key);
      if (y1.length === 0)
        throw new Error("The task has no enabled channels with numeric values.");
      const names = tasks.map(({ name }) => name);
      create({
        name: `Plot for ${strings.naturalLanguageJoin(names, "task")}`,
        channels: { y1 },
      });
    }, "Failed to plot task");
};
