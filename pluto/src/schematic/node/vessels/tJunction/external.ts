// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Theming } from "@synnaxlabs/charon/theming";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Primitive as BasePrimitive } from "@/schematic/node/common/primitive";
import { type Spec } from "@/schematic/node/spec";
import { type Config, VARIANT } from "@/schematic/node/vessels/tJunction/config";
import { Primitive } from "@/schematic/node/vessels/tJunction/Primitive";

export * from "@/schematic/node/vessels/tJunction/config";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  color: t.colors.gray.l11,
  label: Label.defaultConfig(""),
  ...BasePrimitive.ZERO_PROPS,
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "T Junction",
  Form: Form.StyleForm,
  Node: Label.createLabeled<Config>(Primitive),
  Preview: Primitive,
  defaultConfig,
  zIndex: 24,
};
