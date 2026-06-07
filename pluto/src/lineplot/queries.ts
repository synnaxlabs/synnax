// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, NotFoundError, ontology, type workspace } from "@synnaxlabs/client";
import { array, DataType, uuid } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

import { Channel } from "@/channel";
import { Flux } from "@/flux";
import { useSyncedRef } from "@/hooks/ref";
import {
  type DerivedLine,
  type RawDerivedLine,
  resolveLineColor,
} from "@/lineplot/derive";
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
      (r) => onChange(state.skipUndefined((p) => ({ ...p, name: r.name }))),
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
      if (result.variant !== "success") return;
      onChangeRef.current(result.data.name);
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
      lines.map((line, i) => ({
        ...line,
        color: resolveLineColor(line.color, i, palette),
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
  // Keys only change when lines are added, removed, or reordered — not when a
  // line's styling changes — so this stays referentially stable across edits.
  equal: (a, b) => a.length === b.length && a.every((v, i) => v === b[i]),
});

export interface SelectYAxisArgs {
  key: lineplot.Key;
  axisKey: lineplot.YAxisKey;
}

// useSelectYAxisLineKeys returns the keys of the lines plotted on the given
// y-axis. Stable across styling edits (only changes when lines are added or
// removed) so a y-axis re-renders only when its membership changes.
export const useSelectYAxisLineKeys = Flux.createSelector<
  FluxSubStore,
  SelectYAxisArgs,
  string[]
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key, axisKey }) =>
    requireLinePlot(store, key)
      .lines.filter((l) => lineplot.parseLineKey(l.key).yAxis === axisKey)
      .map((l) => l.key),
  equal: (a, b) => a.length === b.length && a.every((v, i) => v === b[i]),
});

export interface SelectXAxisArgs {
  key: lineplot.Key;
  axisKey: lineplot.XAxisKey;
}

// useSelectXAxis returns the x-axis configuration with its tick type resolved:
// a null stored type is derived from the plotted channel's data type (timestamp
// → time, otherwise linear), defaulting to time while the channel loads. A
// non-null stored type is an explicit user override.
export const useSelectXAxis = (args: SelectXAxisArgs): lineplot.Axis => {
  const axis = useSelectAxis({ key: args.key, axisKey: args.axisKey });
  const channelKey = useSelectChannels({ key: args.key })[args.axisKey];
  const { data } = Channel.useRetrieve({ key: channelKey });
  return useMemo(() => {
    if (axis.type != null) return axis;
    const type: lineplot.TickType =
      channelKey === 0 || data == null
        ? "time"
        : data.dataType.equals(DataType.TIMESTAMP)
          ? "time"
          : "linear";
    return { ...axis, type };
  }, [axis, channelKey, data]);
};

export interface YAxisSelection {
  axis: lineplot.Axis;
  lineKeys: string[];
}

// useSelectYAxis returns the y-axis configuration together with the keys of the
// lines plotted on it.
export const useSelectYAxis = (args: SelectYAxisArgs): YAxisSelection => {
  const axis = useSelectAxis({ key: args.key, axisKey: args.axisKey });
  const lineKeys = useSelectYAxisLineKeys(args);
  return useMemo(() => ({ axis, lineKeys }), [axis, lineKeys]);
};

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
const useSelectRawLine = Flux.createSelector<
  FluxSubStore,
  SelectLineArgs,
  RawLine | undefined
>({
  subscribe: (store, { key }, notify) => store.lineplots.onSet(notify, key),
  select: (store, { key, lineKey }) => {
    const lines = store.lineplots.get(key)?.lines;
    if (lines == null) return undefined;
    const index = lines.findIndex((l) => l.key === lineKey);
    return index === -1 ? undefined : { line: lines[index], index };
  },
  equal: (a, b) => a?.line === b?.line && a?.index === b?.index,
});

// useSelectLine returns a single line, enriched with its identity and its color
// resolved by position the same way as useSelectLines, subscribing narrowly so
// it re-renders only when that line, its position, or the palette changes.
export const useSelectLine = (args: SelectLineArgs): DerivedLine | undefined => {
  const raw = useSelectRawLine(args);
  const palette = Theming.use().colors.visualization.palettes.default;
  return useMemo(() => {
    if (raw == null) return undefined;
    return {
      ...raw.line,
      ...lineplot.parseLineKey(raw.line.key),
      color: resolveLineColor(raw.line.color, raw.index, palette),
    };
  }, [raw, palette]);
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

// Drag streams (axis bound, rule position, line style, legend) coalesce
// into one undo entry within the coalesce window. Per-target keys keep
// gestures on different axes/rules/lines from merging with each other —
// dragging axis x1 then axis x2 should be two undo steps, not one.
const kindOfTransaction = (actions: lineplot.Action[]): string => {
  if (actions.length === 0) return "default";
  if (actions.length === 1) {
    const a = actions[0];
    if (a.type === "set_axis") return `set_axis:${a.setAxis.key}`;
    if (a.type === "set_rule") return `set_rule:${a.setRule.rule.key}`;
    if (a.type === "set_line") return `set_line:${a.setLine.key}`;
    return a.type;
  }
  return "transaction";
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
