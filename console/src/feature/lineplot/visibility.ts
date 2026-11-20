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

/** The keys of every range and stored line on the plot. */
export interface Scope {
  ranges: string[];
  lines: string[];
}

export const NONE_HIDDEN: Hidden = { lines: [], ranges: [], traces: [] };

const without = (keys: string[], key: string): string[] =>
  keys.filter((k) => k !== key);

const withKey = (keys: string[], key: string): string[] =>
  keys.includes(key) ? keys : [...keys, key];

const set = (keys: string[], key: string, hidden: boolean): string[] =>
  hidden ? withKey(keys, key) : without(keys, key);

const toggleItem = (
  hidden: Hidden,
  item: Item,
  visible: boolean,
  scope: Scope,
): Hidden => {
  switch (item.variant) {
    case "range":
      return { ...hidden, ranges: set(hidden.ranges, item.range, !visible) };
    case "trace": {
      const key = LinePlot.traceKey(item);
      if (!visible || !hidden.lines.includes(item.line))
        return { ...hidden, traces: set(hidden.traces, key, !visible) };
      // A line hidden in every range shows again only in the range the user picked.
      const others = without(scope.ranges, item.range).map((range) =>
        LinePlot.traceKey({ line: item.line, range }),
      );
      return {
        ...hidden,
        lines: without(hidden.lines, item.line),
        traces: others.reduce(withKey, without(hidden.traces, key)),
      };
    }
    case "line":
      return { ...hidden, lines: set(hidden.lines, item.line, !visible) };
  }
};

const showOnly = (item: Item, scope: Scope): Hidden => {
  switch (item.variant) {
    case "range":
      return { ...NONE_HIDDEN, ranges: without(scope.ranges, item.range) };
    case "trace":
      return {
        ...NONE_HIDDEN,
        ranges: without(scope.ranges, item.range),
        traces: without(scope.lines, item.line).map((line) =>
          LinePlot.traceKey({ line, range: item.range }),
        ),
      };
    case "line":
      return { ...NONE_HIDDEN, lines: without(scope.lines, item.line) };
  }
};

/** What a plot hides, and what a second solo restores. */
export interface Visibility {
  hidden: Hidden;
  soloRestore: Hidden | null;
}

/**
 * @returns state with item shown or hidden. A solo shows only item, and a second solo
 * restores what was hidden before the first.
 */
export const setItemVisible = (
  state: Visibility,
  item: Item,
  visible: boolean,
  solo: boolean,
  scope: Scope,
): Visibility => {
  if (!solo)
    return { ...state, hidden: toggleItem(state.hidden, item, visible, scope) };
  if (state.soloRestore != null)
    return { hidden: state.soloRestore, soloRestore: null };
  return { hidden: showOnly(item, scope), soloRestore: state.hidden };
};

/** @returns the number of items hidden. */
export const countHidden = ({ lines, ranges, traces }: Hidden): number =>
  lines.length + ranges.length + traces.length;
