// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, color } from "@synnaxlabs/x";

import { Component } from "@/component";
import { Border } from "@/schematic/node/common/border";
import { Label } from "@/schematic/node/common/label";
import { type Spec } from "@/schematic/node/spec";
import { type Config, VARIANT } from "@/schematic/node/vessels/tank/config";
import { TankForm } from "@/schematic/node/vessels/tank/Form";
import { Tank } from "@/schematic/node/vessels/tank/Primitive";
import { Symbol } from "@/schematic/node/vessels/tank/Symbol";
import { type Theming } from "@/theming";

export * from "@/schematic/node/vessels/tank/config";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  orientation: "left",
  color: color.ZERO,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  label: Label.defaultConfig("Tank"),
  dimensions: { width: 125, height: 200 },
  borderRadius: Border.DEFAULT_RADIUS,
  fillColor: color.hex(t.colors.visualization.palettes.default[0]),
  bounds: bounds.construct(0, 100),
  showScale: false,
  scaleSide: "left",
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Tank",
  Form: () => <TankForm showFill />,
  Node: Symbol,
  Preview: Component.removeProps(Tank, ["dimensions"]),
  defaultConfig,
  zIndex: 2,
  needsPosition: true,
};
