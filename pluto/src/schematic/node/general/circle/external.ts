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

import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { CircleForm } from "@/schematic/node/general/circle/Form";
import { Circle } from "@/schematic/node/general/circle/Primitive";
import { type Spec } from "@/schematic/node/spec";

const NAME = "Circle";

export const spec: Spec<"circle", schematic.CircleNodeConfig> = {
  key: "circle",
  name: NAME,
  Form: CircleForm,
  Node: Label.createLabeled<schematic.CircleNodeConfig>(Circle, {
    grid: { allowRotate: false, keepAspectRatio: true },
    onResize: ({ width }) => ({ radius: width / (2 * Primitive.BASE_SCALE) }),
  }),
  Preview: Component.removeProps(Circle, ["clickable"]),
  zIndex: 2,
};
