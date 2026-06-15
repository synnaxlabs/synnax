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

import { removeProps } from "@/component/removeProps";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { CircleForm } from "@/schematic/node/general/circle/Form";
import { Circle } from "@/schematic/node/general/circle/Primitive";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

const NAME = "Circle";

export const defaultConfig = (t: Theming.Theme): schematic.NodeConfigCircle => ({
  variant: "circle",
  radius: 20,
  color: color.ZERO,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  label: Label.defaultConfig(NAME),
  strokeWidth: 2,
});

export const spec: Spec<"circle", schematic.NodeConfigCircle> = {
  key: "circle",
  name: NAME,
  Form: CircleForm,
  Node: Label.createLabeled<schematic.NodeConfigCircle>(Circle, {
    grid: { allowRotate: false, keepAspectRatio: true },
    onResize: ({ width }) => ({ radius: width / (2 * Primitive.BASE_SCALE) }),
  }),
  Preview: removeProps(Circle, ["clickable"]),
  defaultConfig,
  zIndex: 2,
};
