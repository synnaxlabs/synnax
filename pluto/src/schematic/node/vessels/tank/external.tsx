// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";

import { type Spec } from "@/schematic/node/spec";
import { TankForm } from "@/schematic/node/vessels/tank/Form";
import { Tank } from "@/schematic/node/vessels/tank/Primitive";
import { Symbol } from "@/schematic/node/vessels/tank/Symbol";

export const spec: Spec<"tank", schematic.TankNodeConfig> = {
  key: "tank",
  name: "Tank",
  Form: () => <TankForm showFillTab />,
  Node: Symbol,
  Preview: Component.removeProps(Tank, ["dimensions"]),
  zIndex: 2,
  needsPosition: true,
};
