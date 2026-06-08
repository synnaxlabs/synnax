// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, NotFoundError, ontology, type workspace } from "@synnaxlabs/client";
import { array, color, compare, DataType, primitive, uuid } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

import { Channel } from "@/channel";
import { Flux } from "@/flux";
import { useSyncedRef } from "@/hooks/ref";
import { Ontology } from "@/ontology";
import { state } from "@/state";
import { Theming } from "@/theming";

export const FLUX_STORE_KEY = "lineplots";
const RESOURCE_NAME = "line plot";

export interface FluxStore extends Flux.UndoableUnaryStore<
  lineplot.Key,
  lineplot.LinePlot,
  lineplot.Action
> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

export type RetrieveQuery = lineplot.RetrieveSingleParams;

export const retrieveSingle = async ({
  store,
  client,
  query: { key },
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.lineplots.get(key);
  if (cached != null) return cached;
  const plot = await client.lineplots.retrieve({ key });
  store.lineplots.set(plot);
  return plot;
};

export const {
  useRetrieve,
  useRetrieveSuspended,
  useRetrieveObservable,
  useEnsureRetrieved,
} = Flux.createRetrieve<RetrieveQuery, lineplot.LinePlot, FluxSubStore>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.lineplots.onSet(onChange, key),
    store.resources.onSet(
      ({ name }) => onChange(state.skipUndefined((p) => ({ ...p, name }))),
      ontology.idToString(lineplot.ontologyID(key)),
    ),
  ],
});

export const useRetrieveObservableName = ({
  onChange,
  ...params
}: Omit<
  Flux.UseRetrieveObservableParams<RetrieveQuery, lineplot.LinePlot>,
  "onChange"
> & {
  onChange: (name: string) => void;
}): Flux.UseRetrieveObservableReturn<RetrieveQuery> => {
  const onChangeRef = useSyncedRef(onChange);
  return useRetrieveObservable({
    ...params,
    onChange: useCallback((result) => {
      if (result.variant === "success") onChangeRef.current(result.data.name);
    }, []),
  });
};

export interface SelectKeyArgs {
  key: lineplot.Key;
}

const requireLinePlot = (store: FluxSubStore, key: lineplot.Key): lineplot.LinePlot => {
  const plot = store.lineplots.get(key);
  if (plot == null) throw new NotFoundError(`Line plot with key ${key} not found`);
  return plot;
};

export const useSelectName = Flux.createSelector<FluxSubStore, SelectKeyArgs, string>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).name,
});

export const useSelectTitle = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  lineplot.Title
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).title,
});

export const useSelectLegend = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  lineplot.Legend
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).legend,
});

export const useSelectChannels = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  lineplot.Channels
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).channels,
});

export const useSelectRanges = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  lineplot.Ranges
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).ranges,
});

export const useSelectAxes = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  lineplot.Axes
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).axes,
});

const shouldDisplayAxis = (
  key: lineplot.AxisKey,
  channels: lineplot.Channels,
): boolean => {
  if (key === "x1" || key === "y1") return true;
  if (key === "x2") return channels.x2 !== 0;
  return channels[key].length > 0;
};

export const useSelectXAxisKeys = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  lineplot.XAxisKey[],
  lineplot.Channels
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).channels,
  transform: (channels) =>
    lineplot.X_AXIS_KEYS.filter((k) => shouldDisplayAxis(k, channels)),
  equal: compare.arraysEqual,
});

export const useSelectYAxisKeys = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  lineplot.YAxisKey[],
  lineplot.Channels
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).channels,
  transform: (channels) =>
    lineplot.Y_AXIS_KEYS.filter((k) => shouldDisplayAxis(k, channels)),
  equal: compare.arraysEqual,
});

export interface SelectAxisArgs {
  key: lineplot.Key;
  axisKey: lineplot.AxisKey;
}

export const useSelectAxis = Flux.createSelector<
  FluxSubStore,
  SelectAxisArgs,
  lineplot.Axis
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key, axisKey }) => requireLinePlot(store, key).axes[axisKey],
});

// RawDerivedLine is a stored line enriched with its decoded identity (axis,
// range, and channel keys parsed from Line.key). Its color may be unset.
export interface RawDerivedLine extends lineplot.Line, lineplot.LineKeyParts {}

// DerivedLine is a RawDerivedLine with its render color resolved to a concrete
// palette color, ready for the chart and toolbar to consume directly.
export interface DerivedLine extends Omit<RawDerivedLine, "color"> {
  color: color.Color;
}

// resolveLineColor returns the concrete color a line should render with: its
// stored color when set, otherwise a palette color chosen by its position. The
// chart and the toolbar both route through this so the displayed colors agree.
const resolveLineColor = (
  stored: color.Color | undefined,
  index: number,
  palette: color.Crude[],
): color.Color =>
  stored ?? color.construct(palette[index % Math.max(palette.length, 1)] ?? color.ZERO);

// useSelectRawLines selects the plot's stored lines enriched with their decoded
// identity. transform is memoized on the stored lines reference, so it only
// re-derives when the lines actually change.
const useSelectRawLines = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  RawDerivedLine[],
  lineplot.Line[]
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).lines,
  transform: (lines) => lines.map((l) => ({ ...l, ...lineplot.parseLineKey(l.key) })),
});

// useSelectLines returns the plot's lines, each enriched with its decoded
// identity and its render color resolved from the active palette (a line with
// no stored color is assigned one by its position). Lines are materialized
// eagerly by the reducer, so this is the complete set of plotted lines.
export const useSelectLines = (args: SelectKeyArgs): DerivedLine[] => {
  const lines = useSelectRawLines(args);
  const palette = Theming.use().colors.visualization.palettes.default;
  return useMemo(
    () =>
      lines.map(({ color, ...line }, i) => ({
        ...line,
        color: resolveLineColor(color, i, palette),
      })),
    [lines, palette],
  );
};

export const useSelectLineKeys = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  string[]
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).lines.map((l) => l.key),
  equal: compare.arraysEqual,
});

export const useSelectLineCount = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  number
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).lines.length,
});

export interface SelectYAxisArgs {
  key: lineplot.Key;
  axisKey: lineplot.YAxisKey;
}

export interface SelectXAxisArgs {
  key: lineplot.Key;
  axisKey: lineplot.XAxisKey;
}

interface SelectXAxisBaseReturn {
  axis: lineplot.Axis;
  channel: lineplot.Channels[lineplot.XAxisKey];
}

const useSelectXAxisBase = Flux.createSelector<
  FluxSubStore,
  SelectXAxisArgs,
  SelectXAxisBaseReturn
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key, axisKey }) => {
    const plot = requireLinePlot(store, key);
    return { axis: plot.axes[axisKey], channel: plot.channels[axisKey] };
  },
  equal: (a, b) => a.axis === b.axis && a.channel === b.channel,
});

// useSelectXAxis returns the x-axis configuration with its tick type resolved:
// a null stored type is derived from the plotted channel's data type (timestamp
// → time, otherwise linear), defaulting to time while the channel loads. A
// non-null stored type is an explicit user override.
export const useSelectXAxis = (args: SelectXAxisArgs): lineplot.Axis => {
  const { axis, channel } = useSelectXAxisBase({
    key: args.key,
    axisKey: args.axisKey,
  });
  const { data: chan } = Channel.useRetrieve(
    { key: channel },
    { beforeRetrieve: ({ query: { key } }) => primitive.isNonZero(key) },
  );
  return useMemo(() => {
    if (axis.type != null) return axis;
    let type: lineplot.TickType = "linear";
    if (channel == 0 || chan == null || chan.dataType.equals(DataType.TIMESTAMP))
      type = "time";
    return { ...axis, type };
  }, [axis, channel, chan]);
};

interface SelectYAxisReturn {
  axis: lineplot.Axis;
  channels: lineplot.Channels[lineplot.YAxisKey];
  lineKeys: string[];
}

export const useSelectYAxis = Flux.createSelector<
  FluxSubStore,
  SelectYAxisArgs,
  SelectYAxisReturn
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key, axisKey }) => {
    const plot = requireLinePlot(store, key);
    const lineKeys = plot.lines
      .filter((l) => lineplot.parseLineKey(l.key).yAxis === axisKey)
      .map((l) => l.key);
    return { axis: plot.axes[axisKey], channels: plot.channels[axisKey], lineKeys };
  },
  equal: (a, b) =>
    a.axis == b.axis &&
    compare.arraysEqual(a.channels, b.channels) &&
    compare.arraysEqual(a.lineKeys, b.lineKeys),
});

export interface SelectLineArgs {
  key: lineplot.Key;
  lineKey: string;
}

interface RawLine {
  line: lineplot.Line;
  index: number;
}

// useSelectRawLine selects a single line and its position by key. The equality
// compares the stored line reference (kept stable across unrelated edits by
// Immer) and the index, so it re-renders only when that line or its position
// changes — not when other lines on the plot do.
const useSelectRawLine = Flux.createSelector<FluxSubStore, SelectLineArgs, RawLine>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key, lineKey }) => {
    const lines = requireLinePlot(store, key).lines;
    const index = lines.findIndex((l) => l.key === lineKey);
    if (index === -1) throw new NotFoundError(`line with key ${lineKey} not found`);
    return { line: lines[index], index };
  },
  equal: (a, b) => a.line === b.line && a.index === b.index,
});

// useSelectLine returns a single line, enriched with its identity and its color
// resolved by position the same way as useSelectLines, subscribing narrowly so
// it re-renders only when that line, its position, or the palette changes.
export const useSelectLine = (args: SelectLineArgs): DerivedLine => {
  const raw = useSelectRawLine(args);
  const palette = Theming.use().colors.visualization.palettes.default;
  const parsed = useMemo(
    () => ({
      ...raw.line,
      ...lineplot.parseLineKey(raw.line.key),
      color: resolveLineColor(raw.line.color, raw.index, palette),
    }),
    [raw, palette],
  );
  const { data: chan } = Channel.useRetrieve({ key: parsed.yChannel });
  if (parsed.label == null && chan != null) parsed.label = chan.name;
  return parsed;
};

export const useSelectRules = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  lineplot.Rule[]
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key }) => requireLinePlot(store, key).rules,
});

export interface SelectRuleArgs {
  key: lineplot.Key;
  ruleKey: string;
}

export const useSelectRule = Flux.createSelector<
  FluxSubStore,
  SelectRuleArgs,
  lineplot.Rule | undefined
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key, ruleKey }) =>
    store.lineplots.get(key)?.rules?.find((r) => r.key === ruleKey),
});

export interface SelectAxisRulesArgs {
  key: lineplot.Key;
  axisKey: lineplot.AxisKey;
}

// useSelectAxisRuleKeys returns the keys of the rules attached to the given axis.
// Stable across edits to individual rules (only changes when rules are added or
// removed) so an axis re-renders only when its rule membership changes.
export const useSelectAxisRuleKeys = Flux.createSelector<
  FluxSubStore,
  SelectAxisRulesArgs,
  string[]
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key, axisKey }) =>
    requireLinePlot(store, key)
      .rules.filter((r) => r.axis === axisKey)
      .map((r) => r.key),
  equal: (a, b) => a.length === b.length && a.every((v, i) => v === b[i]),
});

export type UseDeleteArgs = lineplot.Key | lineplot.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const keys = array.toArray(data);
    const ids = lineplot.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.lineplots.delete(keys));
    await client.lineplots.delete(data);
    return data;
  },
});

export interface CreateParams extends lineplot.New {
  workspace?: workspace.Key;
}

export interface CreateOutput extends lineplot.LinePlot {
  workspace?: workspace.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  CreateParams,
  FluxSubStore,
  CreateOutput
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    data.key ??= uuid.create();
    const { workspace, ...rest } = data;
    rollbacks.push(store.lineplots.set(data.key, data as lineplot.LinePlot));
    const l = await client.lineplots.create(workspace ?? uuid.ZERO, rest);
    store.lineplots.set(l.key, l);
    return { ...l, workspace };
  },
});

export interface RenameParams extends Pick<lineplot.LinePlot, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = data;
    const current = store.lineplots.get(key);
    if (current != null) rollbacks.push(store.lineplots.set(key, { ...current, name }));
    rollbacks.push(Ontology.renameFluxResource(store, lineplot.ontologyID(key), name));
    await client.lineplots.rename(key, name);
    return data;
  },
});

// Drag streams (axis bounds, rule position, line style, legend) coalesce into
// one undo entry within the coalesce window. Single-target actions key by their
// axis/line/rule so gestures on different targets don't merge — dragging axis x1
// then axis x2 is two undo steps. Edits to distinct fields of the same target
// (e.g. an x1 label then its bounds) deliberately share a key, matching the
// previous single-setAxis coalescing.
const kindOfTransaction = (actions: lineplot.Action[]): string => {
  if (actions.length === 0) return "default";
  if (actions.length > 1) return "transaction";
  const a = actions[0];
  switch (a.type) {
    case "set_axis_label":
      return `axis:${a.setAxisLabel.key}`;
    case "set_axis_label_direction":
      return `axis:${a.setAxisLabelDirection.key}`;
    case "set_axis_label_level":
      return `axis:${a.setAxisLabelLevel.key}`;
    case "set_axis_bounds":
      return `axis:${a.setAxisBounds.key}`;
    case "set_axis_tick_spacing":
      return `axis:${a.setAxisTickSpacing.key}`;
    case "set_axis_type":
      return `axis:${a.setAxisType.key}`;
    case "set_line_label":
      return `line:${a.setLineLabel.key}`;
    case "set_line_color":
      return `line:${a.setLineColor.key}`;
    case "set_line_stroke_width":
      return `line:${a.setLineStrokeWidth.key}`;
    case "set_line_downsample":
      return `line:${a.setLineDownsample.key}`;
    case "set_line_downsample_mode":
      return `line:${a.setLineDownsampleMode.key}`;
    case "set_line":
      return `line:${a.setLine.line.key}`;
    case "set_rule":
      return `rule:${a.setRule.rule.key}`;
    case "set_rule_label":
      return `rule:${a.setRuleLabel.key}`;
    case "set_rule_color":
      return `rule:${a.setRuleColor.key}`;
    case "set_rule_axis":
      return `rule:${a.setRuleAxis.key}`;
    case "set_rule_line_width":
      return `rule:${a.setRuleLineWidth.key}`;
    case "set_rule_line_dash":
      return `rule:${a.setRuleLineDash.key}`;
    case "set_rule_units":
      return `rule:${a.setRuleUnits.key}`;
    case "set_rule_position":
      return `rule:${a.setRulePosition.key}`;
    default:
      return a.type;
  }
};

export const FLUX_STORE_CONFIG = Flux.createUndoableStore<
  lineplot.Key,
  lineplot.LinePlot,
  lineplot.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  reduce: lineplot.reduceAll,
  channel: lineplot.SET_CHANNEL_NAME,
  schema: lineplot.scopedActionZ,
  kindOf: kindOfTransaction,
});

export const { useDispatch, useUndo, useRedo } = Flux.createDispatch<
  lineplot.Key,
  lineplot.LinePlot,
  lineplot.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  send: ({ client, key, actions, dispatchKey }) =>
    client.lineplots.dispatch(key, dispatchKey, actions),
});
