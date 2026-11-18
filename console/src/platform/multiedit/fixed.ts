// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/client";
import { color as xcolor } from "@synnaxlabs/x";

// A field that stores color values carries fixed references through the multi-edit
// controls, which hide the palettes for it.

/** @returns the fixed reference to the color. */
export const fixedRef = (value: xcolor.Crude): color.Ref => ({
  type: "fixed",
  value: xcolor.construct(value),
});

/**
 * @returns the color of a fixed reference.
 * @throws if the reference is named, which a control that hides the palettes never
 * sends.
 */
export const fixedValue = (ref: color.Ref): xcolor.Color => {
  if (ref.type === "named")
    throw new Error(`[multiedit] - named color ${ref.key} in a color value field`);
  return ref.value;
};
