// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CommonDummyToggleForm } from "@/schematic/node/common/forms";
import { Label } from "@/schematic/node/common/label";
import { createDummyToggle } from "@/schematic/node/common/symbol/factories";
import { type Spec } from "@/schematic/node/spec";
import { type Config } from "@/schematic/node/valves/breather/config";
import { Primitive } from "@/schematic/node/valves/breather/Primitive";
import { type Theming } from "@/theming";
export type { Config };

export const VARIANT = "breatherValve";
export const NAME = "Breather";
const ZERO_PROPS = { orientation: "left" as const, scale: 1 };

const ZERO_DUMMY_TOGGLE_PROPS = { ...ZERO_PROPS, enabled: false, clickable: false };
export const defaultConfig = (t: Theming.Theme): Config => ({
  color: t.colors.gray.l11,
  label: Label.defaultConfig("Breather Valve"),
  ...ZERO_DUMMY_TOGGLE_PROPS,
});
export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: NAME,
  Form: CommonDummyToggleForm,
  Node: createDummyToggle<Config>(Primitive),
  Preview: Primitive,
  defaultConfig,
  zIndex: 4,
};
