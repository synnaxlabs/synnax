// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, NotFoundError, type project } from "@synnaxlabs/client";
import { color, compare, DataType, type require, uuid, verbs } from "@synnaxlabs/x";
import { useMemo } from "react";

import { Channel } from "@/channel";
import { Flux } from "@/flux";
import { Scope } from "@/lineplot/scope";
import { Theming } from "@/theming";

const RESOURCE_NAME = "line plot";

export type RetrieveQuery = lineplot.RetrieveSingleParams;

export const { use, useEnsure, useTombstone, createSelector } = Flux.createRetrieve<
  RetrieveQuery,
  lineplot.LinePlot
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.lineplots.retrieve(query),
  onChange: ({ client, query }, handler) => client.lineplots.onChange(query, handler),
  getCached: ({ client, query }) => client.lineplots.getCached(query),
  awaitCreation: true,
});

export interface KeyParams {
  key: lineplot.Key;
}

export const useName = Scope.bindHook(createSelector(({ name }) => name));

export const useTitle = Scope.bindHook(createSelector(({ title }) => title));

export const useLegend = Scope.bindHook(createSelector(({ legend }) => legend));

export const useRanges = Scope.bindHook(createSelector(({ ranges }) => ranges));

export const useAxes = Scope.bindHook(createSelector(({ axes }) => axes));

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
  Scope.bindHook(
    createSelector(
      ({ channels }) => keys.filter((k) => shouldDisplayAxis(k, channels)),
      compare.arraysEqual,
    ),
  );

export const useXAxisKeys = createAxisKeysSelector(lineplot.X_AXIS_KEYS);

export const useYAxisKeys = createAxisKeysSelector(lineplot.Y_AXIS_KEYS);

export const useAxisKeys = createAxisKeysSelector(lineplot.AXIS_KEYS);

export interface AxisParams {
  key: lineplot.Key;
  axisKey: lineplot.AxisKey;
}

export const useAxis = Scope.bindHook(
  createSelector<lineplot.Axis, AxisParams>(({ axes }, { axisKey }) => axes[axisKey]),
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

const useStoredLines = createSelector(({ lines }) => lines);

// useRawLines selects the plot's stored lines enriched with their decoded
// identity, memoized on the stored lines reference so it only re-derives when
// the lines actually change.
const useRawLines = (params: KeyParams): RawDerivedLine[] => {
  const lines = useStoredLines(params);
  return useMemo(
    () => lines.map((l) => ({ ...l, ...lineplot.parseLineKey(l.key) })),
    [lines],
  );
};

// useLines returns the plot's lines, each enriched with its decoded
// identity and its render color resolved from the active palette (a line with
// no stored color is assigned one by its position). Lines are materialized
// eagerly by the reducer, so this is the complete set of plotted lines.
export const useLines = Scope.bindHook((params: KeyParams): DerivedLine[] => {
  const lines = useRawLines(params);
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
});

export const useLineKeys = Scope.bindHook(
  createSelector(({ lines }) => lines.map((l) => l.key), compare.arraysEqual),
);

export const useLineCount = Scope.bindHook(createSelector(({ lines }) => lines.length));

export interface YAxisParams {
  key: lineplot.Key;
  axisKey: lineplot.YAxisKey;
}

export interface XAxisParams {
  key: lineplot.Key;
  axisKey: lineplot.XAxisKey;
}

// useYAxisChannels selects the channels plotted on a single y-axis. It
// subscribes at axis granularity, so editing one y-axis's channel set does not
// re-render controls bound to a different axis.
export const useYAxisChannels = Scope.bindHook(
  createSelector<lineplot.Channels[lineplot.YAxisKey], YAxisParams>(
    ({ channels }, { axisKey }) => channels[axisKey],
    compare.arraysEqual,
  ),
);

export const useXAxisChannel = Scope.bindHook(
  createSelector<lineplot.Channels[lineplot.XAxisKey], XAxisParams>(
    ({ channels }, { axisKey }) => channels[axisKey],
  ),
);

export const useXAxisRanges = Scope.bindHook(
  createSelector<lineplot.Ranges[lineplot.XAxisKey], XAxisParams>(
    ({ ranges }, { axisKey }) => ranges[axisKey],
    compare.arraysEqual,
  ),
);

interface XAxisBaseReturn {
  axis: lineplot.Axis;
  channel: lineplot.Channels[lineplot.XAxisKey];
}

const useXAxisBase = createSelector<XAxisBaseReturn, XAxisParams>(
  ({ axes, channels }, { axisKey }) => ({
    axis: axes[axisKey],
    channel: channels[axisKey],
  }),
  (a, b) => a.axis === b.axis && a.channel === b.channel,
);

// useXAxis returns the x-axis configuration with its tick type resolved:
// a null stored type is derived from the plotted channel's data type (timestamp
// maps to time, otherwise linear), defaulting to time while the channel loads.
// A non-null stored type is an explicit user override.
export const useXAxis = Scope.bindHook((params: XAxisParams): lineplot.Axis => {
  const { axis, channel } = useXAxisBase({
    key: params.key,
    axisKey: params.axisKey,
  });
  const { data: dataType } = Channel.useResultDataType(
    channel > 0 ? { key: channel } : null,
  );
  return useMemo(() => {
    if (axis.type != null) return axis;
    let type: lineplot.TickType = "linear";
    if (channel == 0 || dataType == null || dataType.equals(DataType.TIMESTAMP))
      type = "time";
    return { ...axis, type };
  }, [axis, channel, dataType]);
});

interface YAxisReturn {
  axis: lineplot.Axis;
  channels: lineplot.Channels[lineplot.YAxisKey];
  lineKeys: string[];
}

export const useYAxis = Scope.bindHook(
  createSelector<YAxisReturn, YAxisParams>(
    ({ axes, channels, lines }, { axisKey }) => {
      const lineKeys = lines
        .filter((l) => lineplot.parseLineKey(l.key).yAxis === axisKey)
        .map((l) => l.key);
      return { axis: axes[axisKey], channels: channels[axisKey], lineKeys };
    },
    (a, b) =>
      a.axis == b.axis &&
      compare.arraysEqual(a.channels, b.channels) &&
      compare.arraysEqual(a.lineKeys, b.lineKeys),
  ),
);

export interface LineParams {
  key: lineplot.Key;
  lineKey: string;
}

interface RawLine {
  line: lineplot.Line;
  index: number;
}

// useRawLine selects a single line and its position by key. The equality
// compares the stored line reference (kept stable across unrelated edits by
// Immer) and the index, so it re-renders only when that line or its position
// changes, not when other lines on the plot do.
const useRawLine = createSelector<RawLine, LineParams>(
  ({ lines }, { lineKey }) => {
    const index = lines.findIndex((l) => l.key === lineKey);
    if (index === -1) throw new NotFoundError(`line with key ${lineKey} not found`);
    return { line: lines[index], index };
  },
  (a, b) => a.line === b.line && a.index === b.index,
);

// useLine returns a single line, enriched with its identity and its color
// resolved by position the same way as useLines, subscribing narrowly so
// it re-renders only when that line, its position, or the palette changes.
export const useLine = Scope.bindHook(
  (params: LineParams): require.Require<DerivedLine, "label"> => {
    const raw = useRawLine(params);
    const palette = Theming.use().colors.visualization.palettes.default;
    const { yChannel } = lineplot.parseLineKey(raw.line.key);
    const { data: chanName } = Channel.useResultName(
      yChannel > 0 ? { key: yChannel } : null,
    );
    return useMemo(
      () => ({
        ...raw.line,
        ...lineplot.parseLineKey(raw.line.key),
        color: resolvePaletteColor(raw.line.color, raw.index, palette),
        label: raw.line.label ?? chanName ?? "",
        isDefaultLabel: raw.line.label == null,
      }),
      [raw, palette, chanName],
    );
  },
);

// DerivedRule is a stored rule with its render color resolved to a concrete
// palette color, ready for the chart and toolbar to consume directly.
export interface DerivedRule extends Omit<lineplot.Rule, "color"> {
  color: color.Color;
}

const useRawRules = createSelector(({ rules }) => rules);

// useRules returns the plot's rules, each with its render color resolved
// from the active palette (a rule with no stored color is assigned one by its
// position, the same way lines are).
export const useRules = Scope.bindHook((params: KeyParams): DerivedRule[] => {
  const rules = useRawRules(params);
  const palette = Theming.use().colors.visualization.palettes.default;
  return useMemo(
    () =>
      rules.map(({ color, ...rule }, i) => ({
        ...rule,
        color: resolvePaletteColor(color, i, palette),
      })),
    [rules, palette],
  );
});

export interface RuleParams {
  key: lineplot.Key;
  ruleKey: string;
}

interface RawRule {
  rule: lineplot.Rule;
  index: number;
}

const useRawRule = createSelector<RawRule, RuleParams>(
  ({ rules }, { ruleKey }) => {
    const index = rules.findIndex((r) => r.key === ruleKey);
    if (index === -1) throw new NotFoundError(`rule with key ${ruleKey} not found`);
    return { rule: rules[index], index };
  },
  (a, b) => a.rule === b.rule && a.index === b.index,
);

// useRule returns a single rule with its color resolved by position the
// same way as useRules. It throws NotFoundError when no rule with ruleKey
// exists, so callers must only request rules they know are present.
export const useRule = Scope.bindHook((params: RuleParams): DerivedRule => {
  const raw = useRawRule(params);
  const palette = Theming.use().colors.visualization.palettes.default;
  return useMemo(
    () => ({
      ...raw.rule,
      color: resolvePaletteColor(raw.rule.color, raw.index, palette),
    }),
    [raw, palette],
  );
});

export interface AxisRulesParams {
  key: lineplot.Key;
  axisKey: lineplot.AxisKey;
}

// useAxisRuleKeys returns the keys of the rules attached to the given axis.
// Stable across edits to individual rules (only changes when rules are added or
// removed) so an axis re-renders only when its rule membership changes.
export const useAxisRuleKeys = Scope.bindHook(
  createSelector<string[], AxisRulesParams>(
    ({ rules }, { axisKey }) =>
      rules.filter((r) => r.axis === axisKey).map((r) => r.key),
    compare.arraysEqual,
  ),
);

export type UseDeleteParams = lineplot.Key | lineplot.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
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
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.lineplots.create(data.project ?? uuid.ZERO, data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export interface RenameParams extends Pick<lineplot.LinePlot, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
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
