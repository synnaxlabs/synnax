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
import { DEFAULT_BORDER_RADIUS } from "@/schematic/node/common/symbol/border";
import { type Spec } from "@/schematic/node/spec";
import { type Config } from "@/schematic/node/vessels/tank/config";
import { Form } from "@/schematic/node/vessels/tank/Form";
import { Primitive } from "@/schematic/node/vessels/tank/Primitive";
import { Symbol } from "@/schematic/node/vessels/tank/Symbol";
import { type Theming } from "@/theming";

export type { Config };

export const VARIANT = "tank";
export const NAME = "Tank";

export const defaultConfig = (t: Theming.Theme): Config => ({
  orientation: "left",
  color: t.colors.gray.l11,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  label: Label.defaultConfig("Tank"),
  dimensions: { width: 125, height: 200 },
  borderRadius: DEFAULT_BORDER_RADIUS,
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: NAME,
  Form,
  Node: Symbol,
  Preview: Primitive,
  defaultConfig,
  zIndex: 2,
};
