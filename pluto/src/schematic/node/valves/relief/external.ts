// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Theming } from "@synnaxlabs/charon";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Primitive as BasePrimitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import { type Spec } from "@/schematic/node/spec";
import { type Config, VARIANT } from "@/schematic/node/valves/relief/config";
import { Primitive } from "@/schematic/node/valves/relief/Primitive";

export * from "@/schematic/node/valves/relief/config";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  color: t.colors.gray.l11,
  label: Label.defaultConfig("Relief Valve"),
  ...BasePrimitive.ZERO_PROPS,
  ...Toggle.ZERO_DUMMY_TOGGLE_DEFAULTS,
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Relief",
  Form: Form.DummyToggleForm,
  Node: Toggle.createDummyToggle<Config>(Primitive),
  Preview: Primitive,
  defaultConfig,
  zIndex: 4,
};
