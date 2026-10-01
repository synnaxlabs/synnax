// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type theme } from "@synnaxlabs/lyra/theme";
import { type color, type xy } from "@synnaxlabs/x";
import { type FC, type ReactNode } from "react";

import { type Properties } from "@/vis/properties";

export interface FormProps extends Properties.SelectionProps {
  /** actions render at the foot of the form's tab rail. */
  actions?: ReactNode;
  schematicKey?: string;
}

export type NodeProps<Config extends object = object> = {
  nodeKey: string;
  selected: boolean;
  onConfigChange: (data: Partial<Config>) => void;
  config: Config;
  position?: xy.XY;
  draggable?: boolean;
};

export type PreviewProps<P extends object = object> = P & {
  scale?: number;
};

export type Node<Config extends object = object> = FC<NodeProps<Config>>;

/** Resolves the color a symbol paints while one of its colors is absent. */
export type ColorFallback = (theme: theme.Theme) => color.Color;

/** A color field that element forms edit with a per-variant fallback. */
export type ColorKey = "fillColor" | "strokeColor" | "textColor";

/** Maps a color field to the color the element paints while the field is absent. */
export type ColorFallbacks = Partial<Record<ColorKey, ColorFallback>>;

/**
 * Spec describes how to render and edit one node variant. The default parameters give
 * the erased view that holds a spec of any variant.
 */
export interface Spec<
  Variant extends string = schematic.NodeConfigType,
  Config extends object = schematic.NodeConfig,
> {
  key: Variant;
  name: string;
  /** The text written into the symbol's label on creation. Defaults to name. */
  label?: string;
  Form: FC<FormProps>;
  Node: Node<Config>;
  Preview: FC<PreviewProps<Config>>;
  zIndex: number;
  needsPosition?: boolean;
  /** Overrides the default fallback of each color field it lists. */
  colorFallbacks?: ColorFallbacks;
}
