// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Spec } from "@/schematic/node/spec";
import { type Config, VARIANT } from "@/schematic/node/vessels/crossJunction/config";
import { CrossJunction } from "@/schematic/node/vessels/crossJunction/Primitive";
import { type Theming } from "@/theming";

export * from "@/schematic/node/vessels/crossJunction/config";

export const defaultConfig = (_t: Theming.Theme): Config => ({
  variant: VARIANT,
  color: color.ZERO,
  label: Label.defaultConfig(""),
  ...Primitive.ZERO_PROPS,
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Cross Junction",
  Form: Form.StyleForm,
  Node: Label.createLabeled<Config>(CrossJunction),
  Preview: CrossJunction,
  defaultConfig,
  zIndex: 24,
};
