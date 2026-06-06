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
import { type Spec } from "@/schematic/node/spec";
import { type Config, VARIANT } from "@/schematic/node/vessels/cylinder/config";
import { CylinderForm } from "@/schematic/node/vessels/cylinder/Form";
import { Cylinder } from "@/schematic/node/vessels/cylinder/Primitive";
import { Symbol } from "@/schematic/node/vessels/cylinder/Symbol";
import { type Theming } from "@/theming";

export * from "@/schematic/node/vessels/cylinder/config";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  orientation: "left",
  color: color.ZERO,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  label: Label.defaultConfig("Cylinder"),
  dimensions: { width: 66, height: 181 },
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Cylinder",
  Form: CylinderForm,
  Node: Symbol,
  Preview: Cylinder,
  defaultConfig,
  zIndex: 2,
};
