// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, type xy } from "@synnaxlabs/x";

import { type LabeledConfig } from "@/schematic/node/common/symbol/factories";
import { type Value as BaseValue } from "@/vis/value";

export interface Config
  extends LabeledConfig, Omit<BaseValue.UseProps, "box" | "aetherKey"> {
  position?: xy.XY;
  color?: color.Crude;
  textColor?: color.Crude;
  tooltip?: string[];
  redline?: BaseValue.Redline;
  units?: string;
  inlineSize?: number;
}
