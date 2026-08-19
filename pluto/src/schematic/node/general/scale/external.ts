// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Label } from "@/schematic/node/common/label";
import { Scale as BaseScale } from "@/schematic/node/common/scale";
import {
  type Config,
  DEFAULT_DIMENSIONS,
  VARIANT,
} from "@/schematic/node/general/scale/config";
import { ScaleForm } from "@/schematic/node/general/scale/Form";
import { Scale } from "@/schematic/node/general/scale/Primitive";
import { Symbol } from "@/schematic/node/general/scale/Symbol";
import { type Spec } from "@/schematic/node/spec";

export * from "@/schematic/node/general/scale/config";

export const defaultConfig = (): Config => ({
  variant: VARIANT,
  orientation: "left",
  dimensions: { ...DEFAULT_DIMENSIONS },
  label: Label.defaultConfig("Scale"),
  indicator: BaseScale.defaultConfig({ telem: BaseScale.createTelem() }),
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Scale",
  Form: ScaleForm,
  Node: Symbol,
  Preview: Scale,
  defaultConfig,
  zIndex: 4,
  needsPosition: true,
};
