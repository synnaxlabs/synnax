// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { Component } from "@/component";
import { Box } from "@/schematic/node/general/box/Primitive";
import { Symbol } from "@/schematic/node/general/box/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { TankForm } from "@/schematic/node/vessels/tank/Form";

const NAME = "Box";

export const spec: Spec<"box", schematic.BoxNodeConfig> = {
  key: "box",
  name: NAME,
  Form: () => <TankForm showBorderRadius showStrokeWidth />,
  Node: Symbol,
  Preview: Component.removeProps(Box, ["dimensions"]),
  zIndex: 2,
};
