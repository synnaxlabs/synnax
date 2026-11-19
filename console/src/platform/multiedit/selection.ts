// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/client";
import { type color as xcolor, deep, type record } from "@synnaxlabs/x";

import { type ColorRef } from "@/platform/multiedit/config";
import { fixedRef } from "@/platform/multiedit/fixed";

// A control that hides the palettes never sends a named reference.
const fixedColor = (ref: color.Ref): xcolor.Color => {
  if (ref.type === "named")
    throw new Error(`[multiedit] - named color ${ref.key} in a color value field`);
  return ref.value;
};

/**
 * @returns a copy of the config with each path set to its value. An undefined value
 * removes the path, so the field reads as absent.
 */
const patch = <C>(config: C, entries: Array<[string, unknown]>): C => {
  const next = deep.copy(config);
  entries.forEach(([path, value]) => {
    if (value === undefined) deep.remove(next, path);
    else deep.set(next, path, value);
  });
  return next;
};

/** A top-level field that any variant of the config union declares. */
export type Field<C> = C extends unknown ? keyof C & string : never;

/** The value a field holds on the variants that declare it. */
export type Value<C, F extends string> = C extends unknown
  ? F extends keyof C
    ? C[F]
    : never
  : never;

/** A field that holds a color or a color reference on each variant that declares it. */
export type ColorKey<C> = {
  [F in Field<C>]: Value<C, F> extends xcolor.Color | color.Ref | undefined ? F : never;
}[Field<C>];

/** One element's stored color, and the color it paints while the value is absent. */
export interface ColorValue {
  value?: color.Ref;
  fallback: xcolor.Crude;
}

/** One element's config before and after an edit. */
export interface Update<C> {
  key: string;
  base: C;
  next: C;
}

export interface SelectionArgs<C extends { variant: string }> {
  /** The editable configs, by element key. */
  configs: Map<string, C>;
  /** The fields each variant declares, from fieldsByVariant. */
  fields: Map<string, Set<string>>;
  /** The color fields each variant stores as values instead of references. */
  valueColorFields?: Partial<Record<string, Readonly<Record<string, true>>>>;
  /** Called with every changed config. Never called empty. */
  onChange: (updates: Array<Update<C>>) => void;
}

/** Reads and writes one field across every selected element that declares it. */
export interface Selection<C> {
  /** @returns true if any selected element declares the field. */
  has: (field: Field<C>) => boolean;
  /** @returns the value of the field on the first element that declares it. */
  first: <F extends Field<C>>(field: F) => Value<C, F> | undefined;
  /**
   * @returns the stored color of each element that declares the field. A color value
   * reads as a fixed reference.
   */
  colors: (field: ColorKey<C>, fallback: (config: C) => xcolor.Crude) => ColorValue[];
  /** @returns true if an element that declares the field stores it as a color value. */
  valueOnly: (field: ColorKey<C>) => boolean;
  /** Sets the field on every element that declares it. Undefined removes it. */
  set: <F extends Field<C>>(field: F, value: Value<C, F> | undefined) => void;
  /** Sets the field on every element that declares it to fn of its current value. */
  update: <F extends Field<C>>(
    field: F,
    fn: (value: Value<C, F> | undefined) => Value<C, F> | undefined,
  ) => void;
  /**
   * Sets the color field on every element that declares it. An element that stores the
   * field as a value takes the color of the reference. Undefined removes the field.
   * @throws if the reference is named and an element stores the field as a value.
   */
  setColor: (field: ColorKey<C>, value?: color.Ref) => void;
  /**
   * Sets every referenced color to the value.
   * @throws if a reference names an element outside the selection, or if the value is
   * named and a reference is value only.
   */
  setColors: (refs: ColorRef[], value: color.Ref) => void;
}

/**
 * @returns a Selection over the configs.
 * @throws if a config's variant is missing from fields.
 */
export const selection = <C extends { variant: string }>({
  configs,
  fields,
  valueColorFields,
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
  const storesValue = (config: C, field: string): boolean =>
    valueColorFields?.[config.variant]?.[field] === true;
  const readColor = (config: C, field: string): color.Ref | undefined => {
    if (!storesValue(config, field)) return read<color.Ref>(config, field);
    const value = read<xcolor.Color>(config, field);
    return value == null ? undefined : fixedRef(value);
  };
  const apply = (updates: Array<Update<C>>): void => {
    if (updates.length > 0) onChange(updates);
  };
  const update: Selection<C>["update"] = (field, fn) =>
    apply(
      withField(field).map(([key, base]) => ({
        key,
        base,
        next: patch(base, [[field, fn(read(base, field))]]),
      })),
    );
  return {
    has: (field) => withField(field).length > 0,
    first: (field) => {
      const [entry] = withField(field);
      return entry == null ? undefined : read(entry[1], field);
    },
    colors: (field, fallback) =>
      withField(field).map(([, config]) => ({
        value: readColor(config, field),
        fallback: fallback(config),
      })),
    valueOnly: (field) =>
      withField(field).some(([, config]) => storesValue(config, field)),
    set: (field, value) => update(field, () => value),
    update,
    setColor: (field, value) =>
      apply(
        withField(field).map(([key, base]) => ({
          key,
          base,
          next: patch(base, [
            [
              field,
              value != null && storesValue(base, field) ? fixedColor(value) : value,
            ],
          ]),
        })),
      ),
    setColors: (refs, value) =>
      apply(
        Array.from(
          Map.groupBy(refs, (r) => r.key),
          ([key, group]) => {
            const base = configs.get(key);
            if (base == null) throw new Error(`[multiedit] - no config for ${key}`);
            const entries = group.map((r): [string, unknown] => [
              r.path,
              r.valueOnly === true ? fixedColor(value) : value,
            ]);
            return { key, base, next: patch(base, entries) };
          },
        ),
      ),
  };
};
