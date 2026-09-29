// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, type record } from "@synnaxlabs/x";

import { type ColorRef, groupByKey, patch } from "@/platform/multiedit/config";

/** A top-level field that any variant of the config union declares. */
export type Field<C> = C extends unknown ? keyof C & string : never;

/** The value a field holds on the variants that declare it. */
export type Value<C, F extends string> = C extends unknown
  ? F extends keyof C
    ? C[F]
    : never
  : never;

/** A field whose value is a color on every variant that declares it. */
export type ColorKey<C> = {
  [F in Field<C>]: Value<C, F> extends color.Color | undefined ? F : never;
}[Field<C>];

export interface SelectionArgs<C extends { variant: string }> {
  /** The editable configs, by element key. */
  configs: Map<string, C>;
  /** The fields each variant declares, from fieldsByVariant. */
  fields: Map<string, Set<string>>;
  /** Called with every changed config, by element key. Never called empty. */
  onChange: (updates: Array<[string, C]>) => void;
}

/** Reads and writes one field across every selected element that declares it. */
export interface Selection<C> {
  /** @returns true if any selected element declares the field. */
  has: (field: Field<C>) => boolean;
  /** @returns the value of the field on the first element that declares it. */
  first: <F extends Field<C>>(field: F) => Value<C, F> | undefined;
  /** @returns the color of each element that declares the field. */
  colors: (field: ColorKey<C>) => Array<color.Color | undefined>;
  /** Sets the field on every element that declares it. Undefined removes it. */
  set: <F extends Field<C>>(field: F, value: Value<C, F> | undefined) => void;
  /** Sets the field on every element that declares it to fn of its current value. */
  update: <F extends Field<C>>(
    field: F,
    fn: (value: Value<C, F> | undefined) => Value<C, F> | undefined,
  ) => void;
  /**
   * Sets every referenced color to the value.
   * @throws if a reference names an element outside the selection.
   */
  setColors: (refs: ColorRef[], value: color.Color) => void;
}

/**
 * @returns a Selection over the configs.
 * @throws if a config's variant is missing from fields.
 */
export const selection = <C extends { variant: string }>({
  configs,
  fields,
  onChange,
}: SelectionArgs<C>): Selection<C> => {
  const declares = (config: C, field: string): boolean => {
    const declared = fields.get(config.variant);
    if (declared == null)
      throw new Error(`[multiedit] - no schema for ${config.variant}`);
    return declared.has(field);
  };
  const withField = (field: string): Array<[string, C]> =>
    Array.from(configs).filter(([, config]) => declares(config, field));
  const read = <V>(config: C, field: string): V | undefined =>
    (config as record.Unknown)[field] as V | undefined;
  const apply = (updates: Array<[string, C]>): void => {
    if (updates.length > 0) onChange(updates);
  };
  const update: Selection<C>["update"] = (field, fn) =>
    apply(
      withField(field).map(([key, config]) => [
        key,
        patch(config, [[field, fn(read(config, field))]]),
      ]),
    );
  return {
    has: (field) => withField(field).length > 0,
    first: (field) => {
      const [entry] = withField(field);
      return entry == null ? undefined : read(entry[1], field);
    },
    colors: (field) => withField(field).map(([, config]) => read(config, field)),
    set: (field, value) => update(field, () => value),
    update,
    setColors: (refs, value) =>
      apply(
        Array.from(groupByKey(refs), ([key, group]) => {
          const config = configs.get(key);
          if (config == null) throw new Error(`[multiedit] - no config for ${key}`);
          return [
            key,
            patch(
              config,
              group.map((r) => [r.path, value]),
            ),
          ];
        }),
      ),
  };
};
