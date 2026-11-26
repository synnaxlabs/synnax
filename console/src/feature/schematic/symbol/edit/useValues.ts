// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Color, Enumeration, Schematic } from "@synnaxlabs/pluto";
import type { color } from "@synnaxlabs/x";

const path = Schematic.Symbol.formPath;

/** Values holds the values of the state that the editor edits. */
export interface Values {
  /** state is the key of the state that the editor edits. */
  state: string;
  /** states are the states whose values the editor can edit. */
  states: schematic.symbol.EnumerationState[];
  /** list holds the values of the state, named by its enumeration. */
  list: Schematic.Node.Custom.Value[];
  /** entryState is the key of the state when it has an enumeration to color it. */
  entryState?: string;
}

/**
 * @returns the values of the enumeration state key of the symbol in the form. A state
 * takes the entries of its enumeration, or Base and Active when it has none, and each
 * value that a look names. It lists no values while the enumeration loads, or with no
 * state.
 */
export const useValues = (key: string): Values => {
  const all = Form.useFieldValue(path.data.states);
  const looks = Form.useFieldValue(path.data.looks);
  const states = all.filter(
    (s): s is schematic.symbol.EnumerationState => s.type === "enumeration",
  );
  const state = states.find((s) => s.key === key);
  const res = Enumeration.useResult(
    state?.enumeration == null ? null : { key: state.enumeration },
  );
  const entries = res.variant === "success" ? res.data.entries : undefined;
  const listOther = (): Schematic.Node.Custom.Value[] => {
    if (state?.enumeration != null && res.variant === "loading") return [];
    const listed =
      state?.enumeration == null
        ? Schematic.Node.Custom.UNNAMED_VALUES
        : (entries ?? []).map(({ key, value, name }) => ({ key, value, name }));
    const named = looks.flatMap((l) => (l.when?.state === key ? l.when.values : []));
    const extra = [...new Set(named)]
      .filter((v) => !listed.some((l) => l.value === v))
      .map((v) => ({ key: String(v), value: v, name: String(v) }));
    return [...listed, ...extra].sort((a, b) => a.value - b.value);
  };
  const list = state == null ? [] : listOther();
  return {
    state: key,
    states,
    list,
    entryState: state?.enumeration == null ? undefined : key,
  };
};

/**
 * @returns the color that an entry paint on state shows at values, as the preview draws
 * it: the entry for the value of state, or else the entry for its default.
 */
export const useEntryColor = (
  state: schematic.symbol.EnumerationState | undefined,
  values: Schematic.Node.Custom.StateValues,
): color.Color | undefined => {
  const res = Enumeration.useResult(
    state?.enumeration == null ? null : { key: state.enumeration },
  );
  const resolve = Color.useResolve();
  if (state == null || res.variant !== "success") return undefined;
  const value = Schematic.Node.Custom.matched(state, values);
  const { entries } = res.data;
  const entry =
    entries.find((e) => e.value === value) ??
    entries.find((e) => e.value === state.default);
  return entry?.color == null ? undefined : resolve(entry.color);
};
