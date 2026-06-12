// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { z } from "zod";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import { Actuator } from "@/schematic/node/custom/Actuator";
import { Static } from "@/schematic/node/custom/Static";
import { type Spec } from "@/schematic/node/spec";

export const CUSTOM_ACTUATOR_VARIANT = "custom_actuator";

export const customActuatorConfigZ = schematic.nodeConfigCustomActuatorZ.extend({
  stateOverrides: z.array(schematic.symbol.stateZ).optional(),
});
export interface CustomActuatorConfig extends z.infer<typeof customActuatorConfigZ> {}

export const customActuatorSpec: Spec<
  typeof CUSTOM_ACTUATOR_VARIANT,
  CustomActuatorConfig
> = {
  key: CUSTOM_ACTUATOR_VARIANT,
  name: "Custom Actuator",
  Form: Form.ToggleForm,
  Node: Toggle.createToggle<CustomActuatorConfig>(Actuator),
  Preview: Actuator,
  defaultConfig: (): CustomActuatorConfig => ({
    variant: CUSTOM_ACTUATOR_VARIANT,
    specKey: "",
    stateOverrides: [],
    ...Primitive.ZERO_PROPS,
    ...Toggle.ZERO_TOGGLE_DEFAULTS,
    label: Label.defaultConfig("Custom Actuator"),
  }),
  zIndex: 4,
};

export const CUSTOM_STATIC_VARIANT = "custom_static";

export const customStaticConfigZ = schematic.nodeConfigCustomStaticZ.extend({
  stateOverrides: z.array(schematic.symbol.stateZ).optional(),
});
export interface CustomStaticConfig extends z.infer<typeof customStaticConfigZ> {}

export const customStaticSpec: Spec<typeof CUSTOM_STATIC_VARIANT, CustomStaticConfig> =
  {
    key: CUSTOM_STATIC_VARIANT,
    name: "Custom Static",
    Form: Form.StyleForm,
    Node: Label.createLabeled<CustomStaticConfig>(Static),
    Preview: Static,
    defaultConfig: (): CustomStaticConfig => ({
      variant: CUSTOM_STATIC_VARIANT,
      specKey: "",
      stateOverrides: [],
      ...Primitive.ZERO_PROPS,
      label: Label.defaultConfig("Custom Static"),
    }),
    zIndex: 4,
  };
