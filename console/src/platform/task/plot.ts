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
  DisconnectedError,
  type icd,
  type lineplot,
  type Synnax as Client,
  type task,
} from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { DataType, strings, unique } from "@synnaxlabs/x";

import { LinePlot } from "@/platform/lineplot";

/** A channel that a plot of a task shows, with the ICD field that fills it, if any. */
export interface PlotChannel {
  channel: channel.Key;
  /** The field whose units name the axis of the channel. */
  field?: { icd: icd.Key; message: icd.MessageKey; key: icd.FieldKey };
}

/** @returns the channels of a read task's config that a plot of the task shows. */
export type PlotChannels = (config: unknown) => PlotChannel[];

/** An entry of a read task config that fills one channel. */
export interface PlotEntry {
  channel: channel.Key;
  disabled: boolean;
}

/** @returns the channels of the enabled entries. */
export const enabledChannels = (entries: PlotEntry[]): PlotChannel[] =>
  entries.filter(({ disabled }) => !disabled).map(({ channel }) => ({ channel }));

/** @returns the channel that each binding of the reads of an ICD fills. */
export const readChannels = (key: icd.Key, reads: icd.ReadMessage[]): PlotChannel[] =>
  reads.flatMap(({ message, bindings }) =>
    bindings.map(({ field, channel }) => ({
      channel,
      field: { icd: key, message, key: field },
    })),
  );

/** A read task to plot, with the function that finds the channels of its config. */
export interface PlotSource extends Pick<task.Task, "name" | "config"> {
  plotChannels: PlotChannels;
}

const isPlottable = ({ isIndex, dataType }: channel.Channel): boolean =>
  !isIndex && dataType.isNumeric && !dataType.equals(DataType.TIMESTAMP);

/**
 * @returns the trimmed units of the field of each plot channel, or "" for none. A field
 * of a deleted ICD has no units.
 */
const retrieveUnits = async (
  client: Client,
  plotted: PlotChannel[],
): Promise<string[]> => {
  const keys = unique.unique(plotted.flatMap(({ field }) => field?.icd ?? []));
  const icds =
    keys.length === 0
      ? []
      : await client.icds.retrieve({ where: { key: { in: keys } } });
  return plotted.map(({ field }) => {
    if (field == null) return "";
    const message = icds
      .find(({ key }) => key === field.icd)
      ?.messages.find(({ key }) => key === field.message);
    const units = message?.payload.fields.find(({ key }) => key === field.key)?.units;
    return units?.trim() ?? "";
  });
};

const Y_AXES = ["y1", "y2", "y3", "y4"] as const;

/**
 * @returns the y axes of a plot of channels, one for each unit in the order of its
 * first channel. Channels with no units go on y1, and units past the fourth axis share
 * y4. Each axis is labeled with its units.
 */
const createAxes = (
  channels: { key: channel.Key; units: string }[],
): Pick<lineplot.New, "channels" | "axes"> => {
  const seen = unique.unique(channels.map(({ units }) => units));
  const order = seen.includes("") ? ["", ...seen.filter((u) => u !== "")] : seen;
  const axisOf = (units: string) =>
    Y_AXES[Math.min(order.indexOf(units), Y_AXES.length - 1)];
  const used = Y_AXES.slice(0, Math.min(order.length, Y_AXES.length));
  return {
    channels: Object.fromEntries(
      used.map((axis) => [
        axis,
        channels.filter(({ units }) => axisOf(units) === axis).map(({ key }) => key),
      ]),
    ),
    axes: Object.fromEntries(
      used.map((axis) => [
        axis,
        {
          key: axis,
          labelDirection: "y" as const,
          label: order.filter((units) => axisOf(units) === axis).join(", "),
        },
      ]),
    ),
  };
};

/** @returns a function that opens one line plot of the tasks' numeric channels. */
export const usePlot = (): ((tasks: PlotSource[]) => void) => {
  const client = Synnax.use();
  const create = LinePlot.useCreate();
  const handleError = Status.useErrorHandler();
  return (tasks) =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const plotted = unique.by(
        tasks.flatMap(({ config, plotChannels }) => plotChannels(config)),
        ({ channel }) => channel,
      );
      const keys = plotted.map(({ channel }) => channel);
      const channels = keys.length === 0 ? [] : await client.channels.retrieve(keys);
      const plottable = new Set(channels.filter(isPlottable).map(({ key }) => key));
      const shown = plotted.filter(({ channel }) => plottable.has(channel));
      if (shown.length === 0)
        throw new Error("The task has no enabled channels with numeric values.");
      const units = await retrieveUnits(client, shown);
      const names = tasks.map(({ name }) => name);
      create({
        name: `Plot for ${strings.naturalLanguageJoin(names, "task")}`,
        ...createAxes(
          shown.map(({ channel }, i) => ({ key: channel, units: units[i] })),
        ),
      });
    }, "Failed to plot task");
};
