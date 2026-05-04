// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { removeProps } from "@/component/removeProps";
import { Label } from "@/schematic/node/common/label";
import { createLabeled } from "@/schematic/node/common/symbol/factories";
import { type Config } from "@/schematic/node/general/polygon/config";
import { CommonPolygonForm } from "@/schematic/node/general/polygon/Form";
import {
  DEFAULT_POLYGON_SIDE_LENGTH,
  Primitive,
} from "@/schematic/node/general/polygon/Primitive";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export type { Config };

export const VARIANT = "polygon";
export const NAME = "Polygon";

export const defaultConfig = (t: Theming.Theme): Config => ({
  numSides: 6,
  sideLength: DEFAULT_POLYGON_SIDE_LENGTH,
  cornerRounding: 0,
  rotation: 0,
  color: t.colors.gray.l11,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  strokeWidth: 2,
  label: Label.defaultConfig("Polygon"),
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: NAME,
  Form: CommonPolygonForm,
  Node: createLabeled<Config>(Primitive),
  Preview: removeProps(Primitive, ["clickable"]),
  defaultConfig,
  zIndex: 2,
};
