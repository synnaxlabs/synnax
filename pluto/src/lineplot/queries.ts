// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, lineplot, NotFoundError, type project } from "@synnaxlabs/client";
import { array, color as xcolor, DataType, type require, verbs } from "@synnaxlabs/x";

import { Channel } from "@/channel";
import { Color } from "@/color";
import { Flux } from "@/flux";
import { Scope } from "@/lineplot/scope";

const RESOURCE_NAME = "line plot";

export type RetrieveQuery = lineplot.RetrieveSingleParams;

export const { use, useEnsure, useTombstone, createSelector } = Flux.createRetrieve<
  RetrieveQuery,
  lineplot.LinePlot
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.lineplots.retrieve(query.key),
  onChange: ({ client, query }, handler) =>
    client.lineplots.watch(query.key).onChange(handler),
  getCached: ({ client, query }) => client.lineplots.watch(query.key).getCached(),
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
      array.equal,
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

// DerivedLine is a stored line with its render color resolved for the theme, ready for
// the chart and toolbar to consume directly.
export interface DerivedLine extends Omit<lineplot.Line, "color">, ResolvedColor {
  // isDefaultLabel reports whether label is derived from the channel name rather
  // than a stored user override. The toolbar uses it to offer a reset affordance.
  isDefaultLabel: boolean;
}

export interface ResolvedColor {
  // color is the color the element renders with: its stored color when set,
  // otherwise its auto color.
  color: xcolor.Color;
  // pickedColor is the stored color reference, or undefined when the theme picks.
  pickedColor?: color.Ref;
  // autoColor is the series color that the element takes when it has no stored color.
  autoColor: xcolor.Color;
}

interface Colors {
  resolve: (ref: color.Ref) => xcolor.Color | undefined;
  series: color.Key[];
}

const useColors = (): Colors => ({
  resolve: Color.useResolve(),
  series: Color.useSeries(),
});

// autoKeys returns the key of the series color that each element takes when it has no
// stored color: the one that the fewest other elements use, with a tie going to the
// first. Each element with no stored color counts as using the key it takes. The Core
// picks the same keys when it fills in empty colors, so a color does not change when
// the Core saves it.
const autoKeys = (
  stored: (color.Ref | undefined)[],
  series: color.Key[],
): (color.Key | undefined)[] => {
  const uses = new Map<color.Key, number>();
  const count = (key: color.Key): number => uses.get(key) ?? 0;
  const add = (key: color.Key, n: number): void => {
    uses.set(key, count(key) + n);
  };
  for (const ref of stored) if (ref?.type === "named") add(ref.key, 1);
  return stored.map((ref) => {
    const own = ref?.type === "named" ? ref.key : undefined;
    if (own != null) add(own, -1);
    let pick: color.Key | undefined;
    for (const key of series) if (pick == null || count(key) < count(pick)) pick = key;
    if (own != null) add(own, 1);
    else if (ref == null && pick != null) add(pick, 1);
    return pick;
  });
};

// resolveColor resolves a line or rule color. Lines and rules both route through
// it so the chart and the toolbar resolve an element to the same color.
const resolveColor = (
  stored: color.Ref | undefined,
  auto: color.Key | undefined,
  colors: Colors,
): ResolvedColor => {
  const autoColor =
    (auto == null ? undefined : colors.resolve({ type: "named", key: auto })) ??
    xcolor.TRANSPARENT;
  const resolved = stored == null ? undefined : colors.resolve(stored);
  return { color: resolved ?? autoColor, pickedColor: stored, autoColor };
};

const useStoredLines = createSelector(({ lines }) => lines);

const useLinesBase = (params: KeyParams): DerivedLine[] => {
  const lines = useStoredLines(params);
  const colors = useColors();
  const auto = autoKeys(
    lines.map((l) => l.color),
    colors.series,
  );
  return lines.map(({ color, ...line }, i) => ({
    ...line,
    ...resolveColor(color, auto[i], colors),
    isDefaultLabel: line.label == null,
  }));
};

const useRangeColorsBase = (params: KeyParams): xcolor.Color[] => {
  const ranges = useRanges(params);
  const colors = useColors();
  const stored = ranges.map((r) => r.color);
  const auto = autoKeys(stored, colors.series);
  return stored.map((c, i) => resolveColor(c, auto[i], colors).color);
};

/**
 * @returns the color that a relative x axis draws each of the plot's ranges in, in the
 * order of useRanges. A range with no stored color takes a series color.
 */
export const useRangeColors = Scope.bindHook(useRangeColorsBase);

// useLines returns the plot's lines, each with its render color resolved for the
// theme (a line with no stored color takes a series color). Lines are materialized
// eagerly by the reducer, so this is the complete set of styled lines.
export const useLines = Scope.bindHook(useLinesBase);

export const useLineKeys = Scope.bindHook(
  createSelector(({ lines }) => lines.map((l) => l.key), array.equal),
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
    array.equal,
  ),
);

export const useXAxisChannel = Scope.bindHook(
  createSelector<lineplot.Channels[lineplot.XAxisKey], XAxisParams>(
    ({ channels }, { axisKey }) => channels[axisKey],
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

const useResolvedXAxis = (params: XAxisParams): lineplot.Axis => {
  const { axis, channel } = useXAxisBase({
    key: params.key,
    axisKey: params.axisKey,
  });
  const { data: dataType } = Channel.useResultDataType(
    channel > 0 ? { key: channel } : null,
  );
  if (axis.type != null) return axis;
  let type: lineplot.TickType = "linear";
  if (channel == 0 || dataType == null || dataType.equals(DataType.TIMESTAMP))
    type = "time";
  return { ...axis, type };
};

// useXAxis returns the x-axis configuration with its tick type resolved:
// a null stored type is derived from the plotted channel's data type (timestamp
// maps to time, otherwise linear), defaulting to time while the channel loads.
// A non-null stored type is an explicit user override.
export const useXAxis = Scope.bindHook(useResolvedXAxis);

interface YAxisReturn {
  axis: lineplot.Axis;
  channels: lineplot.Channels[lineplot.YAxisKey];
  lineKeys: string[];
}

export const useYAxis = Scope.bindHook(
  createSelector<YAxisReturn, YAxisParams>(
    ({ axes, channels, lines }, { axisKey }) => {
      const lineKeys = lines.filter((l) => l.yAxis === axisKey).map((l) => l.key);
      return { axis: axes[axisKey], channels: channels[axisKey], lineKeys };
    },
    (a, b) =>
      a.axis == b.axis &&
      array.equal(a.channels, b.channels) &&
      array.equal(a.lineKeys, b.lineKeys),
  ),
);

export interface LineParams {
  key: lineplot.Key;
  lineKey: string;
}

interface SeriesParams {
  series: color.Key[];
}

interface RawLine {
  line: lineplot.Line;
  auto?: color.Key;
}

// useRawLine selects a single line and its auto color key. The equality compares the
// stored line reference (kept stable across unrelated edits by Immer) and the key, so
// it re-renders only when that line or its auto color changes, not when other lines
// on the plot do.
const useRawLine = createSelector<RawLine, LineParams & SeriesParams>(
  ({ lines }, { lineKey, series }) => {
    const index = lines.findIndex((l) => l.key === lineKey);
    if (index === -1) throw new NotFoundError(`line with key ${lineKey} not found`);
    const auto = autoKeys(
      lines.map((l) => l.color),
      series,
    )[index];
    return { line: lines[index], auto };
  },
  (a, b) => a.line === b.line && a.auto === b.auto,
);

const useLineBase = (params: LineParams): require.Require<DerivedLine, "label"> => {
  const colors = useColors();
  const raw = useRawLine({ ...params, series: colors.series });
  const { yChannel } = raw.line;
  const { data: chanName } = Channel.useResultName(
    yChannel > 0 ? { key: yChannel } : null,
  );
  return {
    ...raw.line,
    ...resolveColor(raw.line.color, raw.auto, colors),
    label: raw.line.label ?? chanName ?? "",
    isDefaultLabel: raw.line.label == null,
  };
};

// useLine returns a single line with its color resolved the same way as useLines,
// subscribing narrowly so it re-renders only when that line, its auto color, or a
// named color changes.
export const useLine = Scope.bindHook(useLineBase);

// DerivedRule is a stored rule with its render color resolved for the theme, ready
// for the chart and toolbar to consume directly.
export interface DerivedRule extends Omit<lineplot.Rule, "color">, ResolvedColor {}

const useRawRules = createSelector(({ rules }) => rules);

const useRulesBase = (params: KeyParams): DerivedRule[] => {
  const rules = useRawRules(params);
  const colors = useColors();
  const auto = autoKeys(
    rules.map((r) => r.color),
    colors.series,
  );
  return rules.map(({ color, ...rule }, i) => ({
    ...rule,
    ...resolveColor(color, auto[i], colors),
  }));
};

// useRules returns the plot's rules, each with its render color resolved for the
// theme (a rule with no stored color takes a series color, as a line does).
export const useRules = Scope.bindHook(useRulesBase);

export interface RuleParams {
  key: lineplot.Key;
  ruleKey: string;
}

interface RawRule {
  rule: lineplot.Rule;
  auto?: color.Key;
}

const useRawRule = createSelector<RawRule, RuleParams & SeriesParams>(
  ({ rules }, { ruleKey, series }) => {
    const index = rules.findIndex((r) => r.key === ruleKey);
    if (index === -1) throw new NotFoundError(`rule with key ${ruleKey} not found`);
    const auto = autoKeys(
      rules.map((r) => r.color),
      series,
    )[index];
    return { rule: rules[index], auto };
  },
  (a, b) => a.rule === b.rule && a.auto === b.auto,
);

const useRuleBase = (params: RuleParams): DerivedRule => {
  const colors = useColors();
  const raw = useRawRule({ ...params, series: colors.series });
  return { ...raw.rule, ...resolveColor(raw.rule.color, raw.auto, colors) };
};

// useRule returns a single rule with its color resolved the same way as useRules. It
// throws NotFoundError when no rule with ruleKey exists, so callers must only request
// rules they know are present.
export const useRule = Scope.bindHook(useRuleBase);

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
    array.equal,
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
  project: project.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  CreateParams,
  lineplot.LinePlot
>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.lineplots.create(data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export interface RenameParams extends Pick<lineplot.LinePlot, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
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
