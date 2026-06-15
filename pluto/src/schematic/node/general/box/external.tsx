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
import { Label } from "@/schematic/node/common/label";
import { Box } from "@/schematic/node/general/box/Primitive";
import { Symbol } from "@/schematic/node/general/box/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { TankForm } from "@/schematic/node/vessels/tank/Form";
import { type Theming } from "@/theming";

const NAME = "Box";

export const defaultConfig = (t: Theming.Theme): schematic.NodeConfigBox => ({
  variant: "box",
  orientation: "left",
  color: color.ZERO,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  label: Label.defaultConfig(NAME),
  dimensions: { width: 125, height: 200 },
  borderRadius: 3,
  strokeWidth: 2,
});

export const spec: Spec<"box", schematic.NodeConfigBox> = {
  key: "box",
  name: NAME,
  Form: () => <TankForm showBorderRadius showStrokeWidth />,
  Node: Symbol,
  Preview: Component.removeProps(Box, ["dimensions"]),
  defaultConfig,
  zIndex: 2,
};
