// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { type FC } from "react";
import { z } from "zod";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Primitive as BasePrimitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export interface SymbolArgs<V extends string> {
  /// variant is the unique discriminant identifying the symbol in the registry.
  variant: V;
  /// name is the human-readable name shown in the symbols toolbar.
  name: string;
  /// label is the default text rendered beneath the symbol on the canvas. Defaults to
  /// name; provide it only to override the on-canvas label.
  label?: string;
  /// Primitive renders the symbol's SVG; it also serves as the toolbar preview.
  Primitive: FC<BasePrimitive.SVGBasedProps>;
  /// zIndex controls canvas draw order. Defaults to 4.
  zIndex?: number;
}

/// createStatic builds a non-interactive labeled symbol: a styled SVG with a label,
/// color, and scale, edited via the shared StyleForm. This is the most common archetype
/// (meters, fittings, safety, process, vessels).
export const createStatic = <V extends string>({
  variant,
  name,
  label = name,
  Primitive,
  zIndex = 4,
}: SymbolArgs<V>) => {
  const configZ = Label.labeledConfigZ.extend({
    variant: z.literal(variant),
    color: color.crudeZ.optional(),
  });
  type Config = z.infer<typeof configZ>;
  const defaultConfig = (t: Theming.Theme): Config => ({
    variant,
    color: t.colors.gray.l11,
    label: Label.defaultConfig(label),
    ...BasePrimitive.ZERO_PROPS,
  });
  const spec: Spec<V, Config> = {
    key: variant,
    name,
    Form: Form.StyleForm,
    Node: Label.createLabeled<Config>(Primitive),
    Preview: Primitive,
    defaultConfig,
    zIndex,
  };
  return { configZ, spec };
};

interface ToggleArgs<V extends string> extends SymbolArgs<V> {
  /// node selects how the symbol renders. "toggle" (default) renders an interactive
  /// toggle bound to the configured telemetry. "labeled" renders a static labeled
  /// symbol while retaining the toggle telemetry config — used by symbols that carry
  /// control config but do not expose an interactive toggle.
  node?: "toggle" | "labeled";
}

/// createToggle builds an actuator symbol backed by a boolean source/sink with a
/// control chip, edited via the shared ToggleForm (valves, pumps).
export const createToggle = <V extends string>({
  variant,
  name,
  label = name,
  Primitive,
  zIndex = 4,
  node = "toggle",
}: ToggleArgs<V>) => {
  const configZ = Toggle.toggleConfigZ.extend({
    variant: z.literal(variant),
    color: color.crudeZ.optional(),
  });
  type Config = z.infer<typeof configZ>;
  const defaultConfig = (t: Theming.Theme): Config => ({
    variant,
    color: t.colors.gray.l11,
    label: Label.defaultConfig(label),
    ...BasePrimitive.ZERO_PROPS,
    ...Toggle.ZERO_TOGGLE_DEFAULTS,
  });
  const spec: Spec<V, Config> = {
    key: variant,
    name,
    Form: Form.ToggleForm,
    Node:
      node === "labeled"
        ? Label.createLabeled<Config>(Primitive)
        : Toggle.createToggle<Config>(Primitive),
    Preview: Primitive,
    defaultConfig,
    zIndex,
  };
  return { configZ, spec };
};

/// createDummyToggle builds a symbol that toggles its appearance purely from local
/// config (enabled/clickable) without binding to telemetry, edited via the shared
/// DummyToggleForm (manual and relief valves).
export const createDummyToggle = <V extends string>({
  variant,
  name,
  label = name,
  Primitive,
  zIndex = 4,
}: SymbolArgs<V>) => {
  const configZ = Toggle.dummyToggleConfigZ.extend({
    variant: z.literal(variant),
    color: color.crudeZ.optional(),
  });
  type Config = z.infer<typeof configZ>;
  const defaultConfig = (t: Theming.Theme): Config => ({
    variant,
    color: t.colors.gray.l11,
    label: Label.defaultConfig(label),
    ...BasePrimitive.ZERO_PROPS,
    ...Toggle.ZERO_DUMMY_TOGGLE_DEFAULTS,
  });
  const spec: Spec<V, Config> = {
    key: variant,
    name,
    Form: Form.DummyToggleForm,
    Node: Toggle.createDummyToggle<Config>(Primitive),
    Preview: Primitive,
    defaultConfig,
    zIndex,
  };
  return { configZ, spec };
};
