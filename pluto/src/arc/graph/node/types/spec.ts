// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { type FC } from "react";

import { type Theming } from "@/theming";

// PreviewProps are the props passed to a function's Preview, the static rendering shown
// in the stages palette. config holds the type's default parameter values.
export interface PreviewProps<C extends object = object> {
  config: C;
  scale?: number;
}

// SymbolProps are the props passed to a function's on-canvas Symbol. config holds the
// node's stored parameter values (including its "type" discriminant); onConfigChange
// commits a partial update to those values.
export interface SymbolProps<C extends object = object> extends PreviewProps<C> {
  nodeKey?: string;
  position?: xy.XY;
  selected?: boolean;
  draggable?: boolean;
  onConfigChange?: (data: Partial<C>) => void;
}

// Spec describes a single Arc function (graph node type): how it renders on the canvas
// and in the palette, how its parameters are edited, and the default config a new
// instance is seeded with.
export interface Spec<T extends string = string, C extends object = object> {
  // key is the function type, persisted as the "type" discriminant on the node config
  // and matched against the Arc compiler's function templates (e.g. "on", "add").
  key: T;
  name: string;
  Form: FC<{}>;
  Symbol: FC<SymbolProps<C>>;
  // defaultConfig builds the parameter values a freshly added node is seeded with. The
  // returned config includes the "type" discriminant.
  defaultConfig: (t: Theming.Theme) => C;
  Preview: FC<PreviewProps<C>>;
  zIndex: number;
}
