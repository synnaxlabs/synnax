// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { Label } from "@/schematic/node/common/label";
import { type Config, VARIANT } from "@/schematic/node/general/light/config";
import { LightForm } from "@/schematic/node/general/light/Form";
import { Light } from "@/schematic/node/general/light/Primitive";
import { Symbol } from "@/schematic/node/general/light/Symbol";
import { type Spec } from "@/schematic/node/spec";

export * from "@/schematic/node/general/light/config";

export const defaultConfig = (): Config => ({
  variant: VARIANT,
  orientation: "left",
  scale: 1,
  color: color.ZERO,
  label: Label.defaultConfig("Light"),
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Light",
  Form: LightForm,
  Node: Symbol,
  Preview: Light,
  defaultConfig,
  zIndex: 4,
};
