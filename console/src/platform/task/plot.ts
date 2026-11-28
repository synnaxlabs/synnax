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
  lineplot,
  NotFoundError,
  type Synnax as Client,
  type task,
} from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { DataType, errors, strings, unique, uuid } from "@synnaxlabs/x";

import { LinePlot } from "@/platform/lineplot";
import { Panel } from "@/platform/panel";
import { Session } from "@/session";

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
export interface PlotSource extends Pick<task.Task, "key" | "name" | "config"> {
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

const Y_AXES = lineplot.Y_AXIS_KEYS;

/** Where channels go on a plot: the channels each axis gains, and its new label. */
interface Placement {
  channels: Record<lineplot.YAxisKey, channel.Key[]>;
  labels: Partial<Record<lineplot.YAxisKey, string>>;
}

/**
 * @returns where channels go on plot, channels with no units first. A channel goes on
 * an axis with channels whose label lists its units, else on the first axis with no
 * channels, labeled with its units, else on y4, whose label gains its units.
 */
const place = (
  plot: Pick<lineplot.LinePlot, "channels" | "axes">,
  channels: { key: channel.Key; units: string }[],
): Placement => {
  const placement: Placement = {
    channels: { y1: [], y2: [], y3: [], y4: [] },
    labels: {},
  };
  const labelOf = (axis: lineplot.YAxisKey) =>
    placement.labels[axis] ?? plot.axes[axis].label;
  const used = new Set(Y_AXES.filter((axis) => plot.channels[axis].length > 0));
  const sorted = [
    ...channels.filter(({ units }) => units === ""),
    ...channels.filter(({ units }) => units !== ""),
  ];
  for (const { key, units } of sorted) {
    let axis = Y_AXES.find(
      (a) => used.has(a) && labelOf(a).split(", ").includes(units),
    );
    if (axis == null) {
      axis = Y_AXES.find((a) => !used.has(a)) ?? "y4";
      const label = used.has(axis)
        ? [labelOf(axis), units].filter((l) => l !== "").join(", ")
        : units;
      if (label !== labelOf(axis)) placement.labels[axis] = label;
      used.add(axis);
    }
    placement.channels[axis].push(key);
  }
  return placement;
};

const EMPTY_PLOT = {
  channels: lineplot.channelsZ.parse({}),
  axes: lineplot.axesZ.parse({}),
};

/** @returns the line plot with key, or null when it no longer exists. */
const retrieveIfExists = async (
  client: Client,
  key: lineplot.Key,
): Promise<lineplot.LinePlot | null> => {
  try {
    return await client.lineplots.retrieve(key);
  } catch (e) {
    if (NotFoundError.matches(e)) return null;
    throw errors.fromUnknown(e);
  }
};

/**
 * @returns a function that opens the line plot of the tasks' numeric channels. Each
 * unit gets its own y axis. It reopens the plot that it made for the same tasks in the
 * active project, with any channel that the plot lacks added, and creates a plot when
 * there is none.
 */
export const usePlot = (): ((tasks: PlotSource[]) => void) => {
  const client = Synnax.use();
  const create = LinePlot.useCreate();
  const openTab = Panel.useOpenTab();
  const dispatch = Session.useDispatch();
  const getOfTasks = Session.LinePlot.useGetOfTasks();
  const getActiveProject = Session.Project.useGetSelected();
  const handleError = Status.useErrorHandler();
  return (tasks) =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const plotted = unique.by(
        tasks.flatMap(({ config, plotChannels }) => plotChannels(config)),
        ({ channel }) => channel,
      );
      const keys = plotted.map(({ channel }) => channel);
      const retrieved = keys.length === 0 ? [] : await client.channels.retrieve(keys);
      const plottable = new Set(retrieved.filter(isPlottable).map(({ key }) => key));
      const shown = plotted.filter(({ channel }) => plottable.has(channel));
      if (shown.length === 0)
        throw new Error("The task has no enabled channels with numeric values.");
      const units = await retrieveUnits(client, shown);
      const withUnits = shown.map(({ channel }, i) => ({
        key: channel,
        units: units[i],
      }));
      const ofTasks = tasks
        .map(({ key }) => key)
        .sort()
        .join(",");
      // The link is read again until no other run changes it during a retrieve. No
      // await may come between the last read and the create, which puts the plot in the
      // client's cache, where the next run finds it.
      let linked = getOfTasks(ofTasks);
      let found = linked == null ? null : await retrieveIfExists(client, linked);
      while (getOfTasks(ofTasks) !== linked) {
        linked = getOfTasks(ofTasks);
        found = linked == null ? null : await retrieveIfExists(client, linked);
      }
      const plot = found?.project === getActiveProject() ? found : null;
      if (plot == null) {
        const key = uuid.create();
        dispatch(Session.LinePlot.setOfTasks({ tasks: ofTasks, key }));
        const { channels, labels } = place(EMPTY_PLOT, withUnits);
        const names = tasks.map(({ name }) => name);
        create({
          key,
          name: `Plot for ${strings.naturalLanguageJoin(names, "task")}`,
          channels,
          axes: Object.fromEntries(
            Y_AXES.flatMap((axis) => {
              const label = labels[axis];
              return label == null
                ? []
                : [[axis, { key: axis, labelDirection: "y", label }]];
            }),
          ),
        });
        return;
      }
      const present = new Set(Y_AXES.flatMap((axis) => plot.channels[axis]));
      const { channels, labels } = place(
        plot,
        withUnits.filter(({ key }) => !present.has(key)),
      );
      const actions = Y_AXES.flatMap((axis) => {
        const label = labels[axis];
        return [
          ...(label == null ? [] : [lineplot.setAxisLabel({ key: axis, label })]),
          ...channels[axis].map((channel) =>
            lineplot.addChannel({ axisKey: axis, channel }),
          ),
        ];
      });
      if (actions.length > 0) await client.lineplots.dispatch(plot.key, actions);
      openTab({ variant: "resource", resource: lineplot.resourceID(plot.key) });
    }, "Failed to plot task");
};
