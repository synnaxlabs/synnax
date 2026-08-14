// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { removeProps } from "@/component/removeProps";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config, VARIANT } from "@/schematic/node/general/circle/config";
import { CircleForm } from "@/schematic/node/general/circle/Form";
import { Circle } from "@/schematic/node/general/circle/Primitive";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export * from "@/schematic/node/general/circle/config";

const NAME = "Circle";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  radius: 20,
  color: color.ZERO,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  label: Label.defaultConfig(NAME),
  strokeWidth: 2,
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: NAME,
  Form: CircleForm,
  Node: Label.createLabeled<Config>(Circle, {
    grid: { allowRotate: false, keepAspectRatio: true },
    onResize: ({ width }) => ({ radius: width / (2 * Primitive.BASE_SCALE) }),
  }),
  Preview: removeProps(Circle, ["clickable"]),
  defaultConfig,
  zIndex: 2,
};
