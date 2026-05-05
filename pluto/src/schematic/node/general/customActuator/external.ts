// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Primitive as BasePrimitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import { type Config, VARIANT } from "@/schematic/node/general/customActuator/config";
import { Primitive } from "@/schematic/node/general/customActuator/Primitive";
import { type Spec } from "@/schematic/node/spec";

export * from "@/schematic/node/general/customActuator/config";

export const defaultConfig = (): Config => ({
  variant: VARIANT,
  specKey: "",
  stateOverrides: [],
  ...BasePrimitive.ZERO_PROPS,
  ...Toggle.ZERO_TOGGLE_DEFAULTS,
  label: Label.defaultConfig("Custom Actuator"),
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Custom Actuator",
  Form: Form.ToggleForm,
  Node: Toggle.createToggle<Config>(Primitive),
  Preview: Primitive,
  defaultConfig,
  zIndex: 4,
};
