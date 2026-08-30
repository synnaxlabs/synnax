// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";

import { Label } from "@/schematic/node/common/label";
import { Scale as BaseScale } from "@/schematic/node/common/scale";
import { ScaleForm } from "@/schematic/node/general/scale/Form";
import { Scale } from "@/schematic/node/general/scale/Primitive";
import { Symbol } from "@/schematic/node/general/scale/Symbol";
import { type Spec } from "@/schematic/node/spec";

export const defaultConfig = (): schematic.ScaleNodeConfig => ({
  variant: "scale",
  orientation: "left",
  dimensions: { ...BaseScale.DEFAULT_DIMENSIONS },
  color: color.ZERO,
  label: Label.defaultConfig("Scale"),
  indicator: BaseScale.defaultConfig({ channel: 0 }),
});

export const spec: Spec<"scale", schematic.ScaleNodeConfig> = {
  key: "scale",
  name: "Scale",
  Form: ScaleForm,
  Node: Symbol,
  Preview: Scale,
  defaultConfig,
  zIndex: 4,
  needsPosition: true,
};
