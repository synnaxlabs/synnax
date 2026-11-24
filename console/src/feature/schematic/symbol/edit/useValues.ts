// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Color, Enumeration, Schematic } from "@synnaxlabs/pluto";
import type { color } from "@synnaxlabs/x";

import { Spec } from "@/feature/schematic/symbol/spec";

const path = Schematic.Symbol.formPath;

/** Values holds the values of the state that the editor edits. */
export interface Values {
  /** list holds the values that the symbol draws, named by its enumeration. */
  list: Schematic.Node.Custom.Value[];
  /** entryState is the key of the state when it has an enumeration to color it. */
  entryState?: string;
  /**
   * @returns the color of the entry for value, or of the entry for the default value of
   * the state when value has none.
   */
  entryColor: (value: number) => color.Color | undefined;
}

/** @returns the values of the state of the symbol in the form. */
export const useValues = (): Values => {
  const states = Form.useFieldValue(path.data.states);
  const variant = Form.useFieldValue(path.data.variant);
  const key = Spec.enumerationKey({ states });
  const state = states.find((s) => s.key === Spec.STATE);
  const fallback = state?.type === "enumeration" ? state.default : undefined;
  const res = Enumeration.useResult(key == null ? null : { key });
  const resolve = Color.useResolve();
  const entries = res.variant === "success" ? res.data.entries : undefined;
  return {
    list: Schematic.Node.Custom.listValues({ variant }, entries),
    entryState: key == null ? undefined : Spec.STATE,
    entryColor: (value) => {
      const entry =
        entries?.find((e) => e.value === value) ??
        entries?.find((e) => e.value === fallback);
      return entry?.color == null ? undefined : resolve(entry.color);
    },
  };
};
