// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { color } from "@synnaxlabs/client";
import { color as xcolor, type record } from "@synnaxlabs/x";

import type { ColorRef } from "@/platform/multiedit/config";
import type { ColorValue } from "@/platform/multiedit/selection";

// A field that stores fixed colors carries fixed references through the multi-edit
// controls, which hide the palettes for it.

/** @returns the fixed reference to the color. */
export const fixedRef = (value: xcolor.Crude): color.Ref => ({
  type: "fixed",
  value: xcolor.construct(value),
});

/** @returns the values of a field that stores fixed colors, as fixed references. */
export const fixedValues = (values: Array<ColorValue<xcolor.Color>>): ColorValue[] =>
  values.map(({ value, fallback }) => ({
    value: value == null ? undefined : fixedRef(value),
    fallback,
  }));

/**
 * @returns the color of a fixed reference.
 * @throws if the reference is named, which a control that hides the palettes never
 * sends.
 */
export const fixedColor = (ref: color.Ref): xcolor.Color => {
  if (ref.type === "named")
    throw new Error(`[multiedit] - named color ${ref.key} in a fixed color field`);
  return ref.value;
};

/**
 * @returns fixed references to the colors stored at the given top-level fields of the
 * config.
 * @throws if a field holds a value that is not a color.
 */
export const fixedColorRefs = (
  key: string,
  config: record.Unknown,
  fields: readonly string[],
): ColorRef[] =>
  fields.flatMap((path) => {
    const value = config[path];
    if (value == null) return [];
    return [{ key, path, value: fixedRef(xcolor.colorZ.parse(value)) }];
  });
