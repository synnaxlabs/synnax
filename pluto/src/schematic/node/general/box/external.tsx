// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { Component } from "@synnaxlabs/charon";
import { Label } from "@/schematic/node/common/label";
import { type Config, VARIANT } from "@/schematic/node/general/box/config";
import { Primitive } from "@/schematic/node/general/box/Primitive";
import { Symbol } from "@/schematic/node/general/box/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { TankForm } from "@/schematic/node/vessels/tank/Form";
import { type Theming } from "@synnaxlabs/charon";

export * from "@/schematic/node/general/box/config";

const NAME = "Box";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  orientation: "left",
  color: t.colors.gray.l11,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  label: Label.defaultConfig(NAME),
  dimensions: { width: 125, height: 200 },
  borderRadius: 3,
  strokeWidth: 2,
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: NAME,
  Form: () => <TankForm showBorderRadius showStrokeWidth />,
  Node: Symbol,
  Preview: Component.removeProps(Primitive, ["dimensions"]),
  defaultConfig,
  zIndex: 2,
};
