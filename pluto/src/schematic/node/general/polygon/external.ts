// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x/color";
import { Component } from "@synnaxlabs/lyra/component";
import type { Theming } from "@synnaxlabs/lyra/theming";

import { Label } from "@/schematic/node/common/label";
import { type Config, VARIANT } from "@/schematic/node/general/polygon/config";
import { CommonPolygonForm } from "@/schematic/node/general/polygon/Form";
import {
  DEFAULT_POLYGON_SIDE_LENGTH,
  Primitive,
} from "@/schematic/node/general/polygon/Primitive";
import { type Spec } from "@/schematic/node/spec";
export * from "@/schematic/node/general/polygon/config";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  numSides: 6,
  sideLength: DEFAULT_POLYGON_SIDE_LENGTH,
  cornerRounding: 0,
  rotation: 0,
  color: t.colors.gray.l11,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  strokeWidth: 2,
  label: Label.defaultConfig("Polygon"),
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Polygon",
  Form: CommonPolygonForm,
  Node: Label.createLabeled<Config>(Primitive),
  Preview: Component.removeProps(Primitive, ["clickable"]),
  defaultConfig,
  zIndex: 2,
};
