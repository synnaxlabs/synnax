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
import { type Config, VARIANT } from "@/schematic/node/safety/isoBurstDisc/config";
import { Primitive } from "@/schematic/node/safety/isoBurstDisc/Primitive";
import { type Spec } from "@/schematic/node/spec";

export * from "@/schematic/node/safety/isoBurstDisc/config";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  color: t.colors.gray.l11,
  label: Label.defaultConfig("ISO Burst Disc"),
  orientation: "left",
  scale: 1,
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "ISO",
  Form: Form.StyleForm,
  Node: Label.createLabeled<Config>(Primitive),
  Preview: Primitive,
  defaultConfig,
  zIndex: 4,
};
