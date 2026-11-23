// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/placeholder.css";

import type { schematic } from "@synnaxlabs/client";
import type { ReactElement } from "react";

import { CSS } from "@/css";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import type { Spec } from "@/schematic/node/spec";

export const SYMBOL_VARIANT = "symbol";

type Config = schematic.SymbolNodeConfig;

const Box = (): ReactElement => (
  <Primitive.Div className={CSS.B("symbol-placeholder")} />
);

/** Draws the label of a symbol node beside an empty box. SY-6750 replaces it. */
export const symbolSpec: Spec<typeof SYMBOL_VARIANT, Config> = {
  key: SYMBOL_VARIANT,
  name: "Symbol",
  Form: () => null,
  Node: Label.createLabeled<Config>(Box),
  Preview: Box,
  zIndex: 4,
};
