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

import { Spec } from "@/feature/schematic/symbol/spec";

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
  /**
   * @returns the color of the entry for value, or of the entry for the default value of
   * the state when value has none.
   */
  entryColor: (value: number) => color.Color | undefined;
}

/**
 * @returns the values of the enumeration state key of the symbol in the form. With no
 * such state, it edits STATE, or else the first enumeration state. STATE keeps the
 * values of the variant. Another state takes the entries of its enumeration, or 0 and 1
 * as an actuator draws them when it has none, and each value that a look names. It
 * lists no values while the enumeration loads.
 */
export const useValues = (key?: string): Values => {
  const all = Form.useFieldValue(path.data.states);
  const variant = Form.useFieldValue(path.data.variant);
  const looks = Form.useFieldValue(path.data.looks);
  const states = all.filter(
    (s): s is schematic.symbol.EnumerationState => s.type === "enumeration",
  );
  const state =
    states.find((s) => s.key === key) ??
    states.find((s) => s.key === Spec.STATE) ??
    states.at(0);
  const stateKey = state?.key ?? Spec.STATE;
  const res = Enumeration.useResult(
    state?.enumeration == null ? null : { key: state.enumeration },
  );
  const resolve = Color.useResolve();
  const entries = res.variant === "success" ? res.data.entries : undefined;
  const listOther = (): Schematic.Node.Custom.Value[] => {
    if (state?.enumeration != null && res.variant === "loading") return [];
    const listed =
      state?.enumeration == null
        ? Schematic.Node.Custom.listValues({ variant: "actuator" })
        : (entries ?? []).map(({ key, value, name }) => ({ key, value, name }));
    const named = looks.flatMap((l) =>
      l.when?.state === stateKey ? l.when.values : [],
    );
    const extra = [...new Set(named)]
      .filter((v) => !listed.some((l) => l.value === v))
      .map((v) => ({ key: String(v), value: v, name: String(v) }));
    return [...listed, ...extra].sort((a, b) => a.value - b.value);
  };
  const list =
    stateKey === Spec.STATE
      ? Schematic.Node.Custom.listValues({ variant }, entries)
      : listOther();
  return {
    state: stateKey,
    states,
    list,
    entryState: state?.enumeration == null ? undefined : stateKey,
    entryColor: (value) => {
      const entry =
        entries?.find((e) => e.value === value) ??
        entries?.find((e) => e.value === state?.default);
      return entry?.color == null ? undefined : resolve(entry.color);
    },
  };
};
