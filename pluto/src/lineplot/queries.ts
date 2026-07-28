// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  lineplot,
  NotFoundError,
  type project,
  query,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { color, compare, DataType, primitive, type require, uuid } from "@synnaxlabs/x";
import { useMemo } from "react";

import { Channel } from "@/channel";
import { Flux } from "@/flux";
import { Scope } from "@/lineplot/scope";
import { Theming } from "@/theming";

const RESOURCE_NAME = "line plot";

export type RetrieveQuery = lineplot.RetrieveSingleParams;

export const {
  useRetrieve,
  useRetrieveSuspended,
  useRetrieveObservable,
  useEnsureRetrieved,
} = Flux.createRetrieve<RetrieveQuery, lineplot.LinePlot>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.lineplots.retrieve(query),
  subscribe: ({ client, query }, handler) => client.lineplots.onChange(query, handler),
  getCached: ({ client, query }) => client.lineplots.getCached(query),
});

export interface SelectKeyParams {
  key: lineplot.Key;
}

const requireLinePlot = (
  client: Client | null,
  key: lineplot.Key,
): lineplot.LinePlot => {
  const cached = client?.lineplots.getCached({ key });
  if (cached == null || query.Deleted.matches(cached))
    throw new NotFoundError(`Line plot with key ${key} not found`);
  return cached;
};

const subscribe = (
  { client, args: { key } }: Flux.SelectorParams<SelectKeyParams>,
  notify: () => void,
) => (client == null ? () => {} : client.lineplots.onChange({ key }, notify));

export const [useSelectName, useGetName] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, string>({
    subscribe,
    select: ({ client, args: { key } }) => requireLinePlot(client, key).name,
  }),
);

export const [useSelectTitle, useGetTitle] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, lineplot.Title>({
    subscribe,
    select: ({ client, args: { key } }) => requireLinePlot(client, key).title,
  }),
);

export const [useSelectLegend, useGetLegend] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, lineplot.Legend>({
    subscribe,
    select: ({ client, args: { key } }) => requireLinePlot(client, key).legend,
  }),
);

export const [useSelectRanges, useGetRanges] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, lineplot.Ranges>({
    subscribe,
    select: ({ client, args: { key } }) => requireLinePlot(client, key).ranges,
  }),
);

export const [useSelectAxes, useGetAxes] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, lineplot.Axes>({
    subscribe,
    select: ({ client, args: { key } }) => requireLinePlot(client, key).axes,
  }),
);

const shouldDisplayAxis = (
  key: lineplot.AxisKey,
  channels: lineplot.Channels,
): boolean => {
  if (key === "x1" || key === "y1") return true;
  if (key === "x2") return channels.x2 !== 0;
  return channels[key].length > 0;
};

// createAxisKeysSelector builds a selector returning the subset of keys whose
// axes should currently be displayed (see shouldDisplayAxis). The result is
// referentially stable until that subset changes, so consumers re-render only
// on membership changes rather than on every channel edit.
const createAxisKeysSelector = <K extends lineplot.AxisKey>(keys: readonly K[]) =>
  Scope.bindSelector(
    Flux.createSelector<SelectKeyParams, K[], lineplot.Channels>({
      subscribe,
      select: ({ client, args: { key } }) => requireLinePlot(client, key).channels,
      transform: (channels) => keys.filter((k) => shouldDisplayAxis(k, channels)),
      equal: compare.arraysEqual,
    }),
  );

export const [useSelectXAxisKeys, useGetXAxisKeys] = createAxisKeysSelector(
  lineplot.X_AXIS_KEYS,
);

export const [useSelectYAxisKeys, useGetYAxisKeys] = createAxisKeysSelector(
  lineplot.Y_AXIS_KEYS,
);

export const [useSelectAxisKeys, useGetAxisKeys] = createAxisKeysSelector(
  lineplot.AXIS_KEYS,
);

export interface SelectAxisParams {
  key: lineplot.Key;
  axisKey: lineplot.AxisKey;
}

export const [useSelectAxis, useGetAxis] = Scope.bindSelector(
  Flux.createSelector<SelectAxisParams, lineplot.Axis>({
    subscribe,
    select: ({ client, args: { key, axisKey } }) =>
      requireLinePlot(client, key).axes[axisKey],
  }),
);

// RawDerivedLine is a stored line enriched with its decoded identity (axis,
// range, and channel keys parsed from Line.key). Its color may be unset.
export interface RawDerivedLine extends lineplot.Line, lineplot.LineKeyParts {}

// DerivedLine is a RawDerivedLine with its render color resolved to a concrete
// palette color, ready for the chart and toolbar to consume directly.
export interface DerivedLine extends Omit<RawDerivedLine, "color"> {
  color: color.Color;
  // isDefaultLabel reports whether label is derived from the channel name rather
  // than a stored user override. The toolbar uses it to offer a reset affordance.
  isDefaultLabel: boolean;
}

// resolvePaletteColor returns the concrete color an element should render with:
// its stored color when set, otherwise a palette color chosen by its position.
// Lines and rules both route through this so their displayed colors agree, and
// so the chart and the toolbar resolve an element to the same color.
const resolvePaletteColor = (
  stored: color.Color | undefined,
  index: number,
  palette: color.Crude[],
): color.Color =>
  stored ?? color.construct(palette[index % Math.max(palette.length, 1)] ?? color.ZERO);

// useSelectRawLines selects the plot's stored lines enriched with their decoded
// identity. transform is memoized on the stored lines reference, so it only
// re-derives when the lines actually change.
const [useSelectRawLines] = Flux.createSelector<
  SelectKeyParams,
  RawDerivedLine[],
  lineplot.Line[]
>({
  subscribe,
  select: ({ client, args: { key } }) => requireLinePlot(client, key).lines,
  transform: (lines) => lines.map((l) => ({ ...l, ...lineplot.parseLineKey(l.key) })),
});

// useSelectLines returns the plot's lines, each enriched with its decoded
// identity and its render color resolved from the active palette (a line with
// no stored color is assigned one by its position). Lines are materialized
// eagerly by the reducer, so this is the complete set of plotted lines.
export const useSelectLines = Scope.bindHook(
  (params: SelectKeyParams): DerivedLine[] => {
    const lines = useSelectRawLines(params);
    const palette = Theming.use().colors.visualization.palettes.default;
    return useMemo(
      () =>
        lines.map(({ color, ...line }, i) => ({
          ...line,
          color: resolvePaletteColor(color, i, palette),
          isDefaultLabel: line.label == null,
        })),
      [lines, palette],
    );
  },
);

export const [useSelectLineKeys, useGetLineKeys] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, string[]>({
    subscribe,
    select: ({ client, args: { key } }) =>
      requireLinePlot(client, key).lines.map((l) => l.key),
    equal: compare.arraysEqual,
  }),
);

export const [useSelectLineCount, useGetLineCount] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, number>({
    subscribe,
    select: ({ client, args: { key } }) => requireLinePlot(client, key).lines.length,
  }),
);

export interface SelectYAxisParams {
  key: lineplot.Key;
  axisKey: lineplot.YAxisKey;
}

export interface SelectXAxisParams {
  key: lineplot.Key;
  axisKey: lineplot.XAxisKey;
}

// useSelectYAxisChannels selects the channels plotted on a single y-axis. It
// subscribes at axis granularity, so editing one y-axis's channel set does not
// re-render controls bound to a different axis.
export const [useSelectYAxisChannels, useGetYAxisChannels] = Scope.bindSelector(
  Flux.createSelector<SelectYAxisParams, lineplot.Channels[lineplot.YAxisKey]>({
    subscribe,
    select: ({ client, args: { key, axisKey } }) =>
      requireLinePlot(client, key).channels[axisKey],
    equal: compare.arraysEqual,
  }),
);

// useSelectXAxisChannel selects the single channel plotted on an x-axis.
export const [useSelectXAxisChannel, useGetXAxisChannel] = Scope.bindSelector(
  Flux.createSelector<SelectXAxisParams, lineplot.Channels[lineplot.XAxisKey]>({
    subscribe,
    select: ({ client, args: { key, axisKey } }) =>
      requireLinePlot(client, key).channels[axisKey],
  }),
);

// useSelectXAxisRanges selects the range keys plotted against an x-axis.
export const [useSelectXAxisRanges, useGetXAxisRanges] = Scope.bindSelector(
  Flux.createSelector<SelectXAxisParams, lineplot.Ranges[lineplot.XAxisKey]>({
    subscribe,
    select: ({ client, args: { key, axisKey } }) =>
      requireLinePlot(client, key).ranges[axisKey],
    equal: compare.arraysEqual,
  }),
);

interface SelectXAxisBaseReturn {
  axis: lineplot.Axis;
  channel: lineplot.Channels[lineplot.XAxisKey];
}

const [useSelectXAxisBase] = Flux.createSelector<
  SelectXAxisParams,
  SelectXAxisBaseReturn
>({
  subscribe,
  select: ({ client, args: { key, axisKey } }) => {
    const plot = requireLinePlot(client, key);
    return { axis: plot.axes[axisKey], channel: plot.channels[axisKey] };
  },
  equal: (a, b) => a.axis === b.axis && a.channel === b.channel,
});

// useSelectXAxis returns the x-axis configuration with its tick type resolved:
// a null stored type is derived from the plotted channel's data type (timestamp
// maps to time, otherwise linear), defaulting to time while the channel loads.
// A non-null stored type is an explicit user override.
export const useSelectXAxis = Scope.bindHook(
  (params: SelectXAxisParams): lineplot.Axis => {
    const { axis, channel } = useSelectXAxisBase({
      key: params.key,
      axisKey: params.axisKey,
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
  },
);

interface SelectYAxisReturn {
  axis: lineplot.Axis;
  channels: lineplot.Channels[lineplot.YAxisKey];
  lineKeys: string[];
}

export const [useSelectYAxis, useGetYAxis] = Scope.bindSelector(
  Flux.createSelector<SelectYAxisParams, SelectYAxisReturn>({
    subscribe,
    select: ({ client, args: { key, axisKey } }) => {
      const plot = requireLinePlot(client, key);
      const lineKeys = plot.lines
        .filter((l) => lineplot.parseLineKey(l.key).yAxis === axisKey)
        .map((l) => l.key);
      return { axis: plot.axes[axisKey], channels: plot.channels[axisKey], lineKeys };
    },
    equal: (a, b) =>
      a.axis == b.axis &&
      compare.arraysEqual(a.channels, b.channels) &&
      compare.arraysEqual(a.lineKeys, b.lineKeys),
  }),
);

export interface SelectLineParams {
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
// changes, not when other lines on the plot do.
const [useSelectRawLine] = Flux.createSelector<SelectLineParams, RawLine>({
  subscribe,
  select: ({ client, args: { key, lineKey } }) => {
    const lines = requireLinePlot(client, key).lines;
    const index = lines.findIndex((l) => l.key === lineKey);
    if (index === -1) throw new NotFoundError(`line with key ${lineKey} not found`);
    return { line: lines[index], index };
  },
  equal: (a, b) => a.line === b.line && a.index === b.index,
});

// useSelectLine returns a single line, enriched with its identity and its color
// resolved by position the same way as useSelectLines, subscribing narrowly so
// it re-renders only when that line, its position, or the palette changes.
export const useSelectLine = Scope.bindHook(
  (params: SelectLineParams): require.Require<DerivedLine, "label"> => {
    const raw = useSelectRawLine(params);
    const palette = Theming.use().colors.visualization.palettes.default;
    const { yChannel } = lineplot.parseLineKey(raw.line.key);
    const { data: chan } = Channel.useRetrieve({ key: yChannel });
    return useMemo(
      () => ({
        ...raw.line,
        ...lineplot.parseLineKey(raw.line.key),
        color: resolvePaletteColor(raw.line.color, raw.index, palette),
        label: raw.line.label ?? chan?.name ?? "",
        isDefaultLabel: raw.line.label == null,
      }),
      [raw, palette, chan],
    );
  },
);

// DerivedRule is a stored rule with its render color resolved to a concrete
// palette color, ready for the chart and toolbar to consume directly.
export interface DerivedRule extends Omit<lineplot.Rule, "color"> {
  color: color.Color;
}

const [useSelectRawRules] = Flux.createSelector<SelectKeyParams, lineplot.Rule[]>({
  subscribe,
  select: ({ client, args: { key } }) => requireLinePlot(client, key).rules,
});

// useSelectRules returns the plot's rules, each with its render color resolved
// from the active palette (a rule with no stored color is assigned one by its
// position, the same way lines are).
export const useSelectRules = Scope.bindHook(
  (params: SelectKeyParams): DerivedRule[] => {
    const rules = useSelectRawRules(params);
    const palette = Theming.use().colors.visualization.palettes.default;
    return useMemo(
      () =>
        rules.map(({ color, ...rule }, i) => ({
          ...rule,
          color: resolvePaletteColor(color, i, palette),
        })),
      [rules, palette],
    );
  },
);

export interface SelectRuleParams {
  key: lineplot.Key;
  ruleKey: string;
}

interface RawRule {
  rule: lineplot.Rule;
  index: number;
}

const [useSelectRawRule] = Flux.createSelector<SelectRuleParams, RawRule>({
  subscribe,
  select: ({ client, args: { key, ruleKey } }) => {
    const rules = requireLinePlot(client, key).rules;
    const index = rules.findIndex((r) => r.key === ruleKey);
    if (index === -1) throw new NotFoundError(`rule with key ${ruleKey} not found`);
    return { rule: rules[index], index };
  },
  equal: (a, b) => a?.rule === b?.rule && a?.index === b?.index,
});

// useSelectRule returns a single rule with its color resolved by position the
// same way as useSelectRules. It throws NotFoundError when no rule with ruleKey
// exists, so callers must only request rules they know are present.
export const useSelectRule = Scope.bindHook((params: SelectRuleParams): DerivedRule => {
  const raw = useSelectRawRule(params);
  const palette = Theming.use().colors.visualization.palettes.default;
  return useMemo(
    () => ({
      ...raw.rule,
      color: resolvePaletteColor(raw.rule.color, raw.index, palette),
    }),
    [raw, palette],
  );
});

export interface SelectAxisRulesParams {
  key: lineplot.Key;
  axisKey: lineplot.AxisKey;
}

// useSelectAxisRuleKeys returns the keys of the rules attached to the given axis.
// Stable across edits to individual rules (only changes when rules are added or
// removed) so an axis re-renders only when its rule membership changes.
export const [useSelectAxisRuleKeys, useGetAxisRuleKeys] = Scope.bindSelector(
  Flux.createSelector<SelectAxisRulesParams, string[]>({
    subscribe,
    select: ({ client, args: { key, axisKey } }) =>
      requireLinePlot(client, key)
        .rules.filter((r) => r.axis === axisKey)
        .map((r) => r.key),
    equal: (a, b) => a.length === b.length && a.every((v, i) => v === b[i]),
  }),
);

export type UseDeleteParams = lineplot.Key | lineplot.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteParams>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.lineplots.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface CreateParams extends lineplot.New {
  project?: project.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  CreateParams,
  lineplot.LinePlot
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.lineplots.create(data.project ?? uuid.ZERO, data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export interface RenameParams extends Pick<lineplot.LinePlot, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await onOptimisticComplete(data);
    await client.lineplots.rename(key, name);
    return data;
  },
});

export const {
  useDispatch,
  useUndo: useUndoBase,
  useRedo: useRedoBase,
  useSingleDispatch: useSingleDispatchBase,
} = Flux.createDispatch<lineplot.Key, lineplot.LinePlot, lineplot.Action>({
  domain: (client) => client.lineplots,
});

export const useSingleDispatch = Scope.bindHook(useSingleDispatchBase);
export const useUndo = Scope.bindHook(useUndoBase);
export const useRedo = Scope.bindHook(useRedoBase);
