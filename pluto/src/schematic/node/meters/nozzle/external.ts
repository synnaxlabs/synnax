// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CommonStyleForm } from "@/schematic/node/common/forms";
import { Label } from "@/schematic/node/common/label";
import { createLabeled } from "@/schematic/node/common/symbol/factories";
import { type Config } from "@/schematic/node/meters/nozzle/config";
import { Primitive } from "@/schematic/node/meters/nozzle/Primitive";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";
export type { Config };

export const VARIANT = "flowmeterNozzle";
export const NAME = "Nozzle";
const ZERO_PROPS = { orientation: "left" as const, scale: 1 };
export const defaultConfig = (t: Theming.Theme): Config => ({
  color: t.colors.gray.l11,
  label: Label.defaultConfig("Nozzle Flowmeter"),
  ...ZERO_PROPS,
});
export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: NAME,
  Form: CommonStyleForm,
  Node: createLabeled<Config>(Primitive),
  Preview: Primitive,
  defaultConfig,
  zIndex: 4,
};
