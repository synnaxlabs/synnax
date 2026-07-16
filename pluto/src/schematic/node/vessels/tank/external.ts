// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";

import { Component } from "@/component";
import { Border } from "@/schematic/node/common/border";
import { Label } from "@/schematic/node/common/label";
import { type Spec } from "@/schematic/node/spec";
import { TankForm } from "@/schematic/node/vessels/tank/Form";
import { Tank } from "@/schematic/node/vessels/tank/Primitive";
import { Symbol } from "@/schematic/node/vessels/tank/Symbol";
import { type Theming } from "@/theming";

export const defaultConfig = (t: Theming.Theme): schematic.NodeConfigTank => ({
  variant: "tank",
  orientation: "left",
  color: color.ZERO,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  label: Label.defaultConfig("Tank"),
  dimensions: { width: 125, height: 200 },
  borderRadius: Border.DEFAULT_RADIUS,
});

export const spec: Spec<"tank", schematic.NodeConfigTank> = {
  key: "tank",
  name: "Tank",
  Form: TankForm,
  Node: Symbol,
  Preview: Component.removeProps(Tank, ["dimensions"]),
  defaultConfig,
  zIndex: 2,
};
