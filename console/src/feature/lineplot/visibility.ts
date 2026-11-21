// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinePlot } from "@synnaxlabs/pluto";

import type { Session } from "@/session";

type Hidden = Session.LinePlot.Hidden;

/** An item of a relative plot's legend that the user can show or hide. */
export type Item =
  | { variant: "range"; range: string }
  | { variant: "trace"; line: string; range: string }
  | { variant: "line"; line: string };

/** The keys of every range a relative axis can draw, and of every stored line. */
export interface Scope {
  ranges: string[];
  lines: string[];
}

/** What a plot hides, and the item a solo showed alone. */
export interface Visibility {
  hidden: Hidden;
  solo: Session.LinePlot.Solo | null;
}

export const NONE_HIDDEN: Hidden = { lines: [], ranges: [], traces: [] };

const without = (keys: string[], key: string): string[] =>
  keys.filter((k) => k !== key);

const withKey = (keys: string[], key: string): string[] =>
  keys.includes(key) ? keys : [...keys, key];

const set = (keys: string[], key: string, hidden: boolean): string[] =>
  hidden ? withKey(keys, key) : without(keys, key);

const tracesOf = (lines: string[], ranges: string[]): string[] =>
  lines.flatMap((line) => ranges.map((range) => LinePlot.traceKey({ line, range })));

const itemKey = (item: Item): string => {
  switch (item.variant) {
    case "range":
      return `range:${item.range}`;
    case "trace":
      return `trace:${LinePlot.traceKey(item)}`;
    case "line":
      return `line:${item.line}`;
  }
};

/**
 * @returns hidden without the keys of ranges and lines outside of scope, such as a
 * range the user removed from the plot.
 */
export const inScope = (hidden: Hidden, { ranges, lines }: Scope): Hidden => {
  const traces = new Set(tracesOf(lines, ranges));
  return {
    lines: hidden.lines.filter((l) => lines.includes(l)),
    ranges: hidden.ranges.filter((r) => ranges.includes(r)),
    traces: hidden.traces.filter((t) => traces.has(t)),
  };
};

// showTrace shows a line in one range. A line hidden in every range shows again only in
// that range, and a hidden range shows again with only that line.
const showTrace = (
  hidden: Hidden,
  line: string,
  range: string,
  scope: Scope,
): Hidden => {
  const traces = without(hidden.traces, LinePlot.traceKey({ line, range }));
  const lineHidden = hidden.lines.includes(line);
  const rangeHidden = hidden.ranges.includes(range);
  const others = [
    ...(lineHidden ? tracesOf([line], without(scope.ranges, range)) : []),
    ...(rangeHidden ? tracesOf(without(scope.lines, line), [range]) : []),
  ];
  return {
    lines: lineHidden ? without(hidden.lines, line) : hidden.lines,
    ranges: rangeHidden ? without(hidden.ranges, range) : hidden.ranges,
    traces: others.reduce(withKey, traces),
  };
};

const only = (item: Item, scope: Scope): Hidden => {
  switch (item.variant) {
    case "range":
      return { ...NONE_HIDDEN, ranges: without(scope.ranges, item.range) };
    case "trace":
      return {
        ...NONE_HIDDEN,
        ranges: without(scope.ranges, item.range),
        traces: tracesOf(without(scope.lines, item.line), [item.range]),
      };
    case "line":
      return { ...NONE_HIDDEN, lines: without(scope.lines, item.line) };
  }
};

/** @returns state with item shown or hidden. */
export const setItemVisible = (
  state: Visibility,
  item: Item,
  visible: boolean,
  scope: Scope,
): Visibility => {
  const { hidden } = state;
  switch (item.variant) {
    case "range":
      return {
        ...state,
        hidden: { ...hidden, ranges: set(hidden.ranges, item.range, !visible) },
      };
    case "trace":
      return {
        ...state,
        hidden: visible
          ? showTrace(hidden, item.line, item.range, scope)
          : { ...hidden, traces: withKey(hidden.traces, LinePlot.traceKey(item)) },
      };
    case "line":
      return {
        ...state,
        hidden: { ...hidden, lines: set(hidden.lines, item.line, !visible) },
      };
  }
};

/**
 * @returns state that shows only item. It keeps what was hidden before the first solo,
 * so that a second solo of the same item restores it.
 */
export const showOnly = (state: Visibility, item: Item, scope: Scope): Visibility => ({
  hidden: only(item, scope),
  solo: { item: itemKey(item), restore: state.solo?.restore ?? state.hidden },
});

/**
 * @returns state that shows only item, or the state from before the solo when item is
 * the one the solo showed alone.
 */
export const toggleSolo = (state: Visibility, item: Item, scope: Scope): Visibility =>
  state.solo?.item === itemKey(item)
    ? { hidden: inScope(state.solo.restore, scope), solo: null }
    : showOnly(state, item, scope);

/** @returns the number of items hidden. */
export const countHidden = ({ lines, ranges, traces }: Hidden): number =>
  lines.length + ranges.length + traces.length;
