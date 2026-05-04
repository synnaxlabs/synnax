// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, type dimensions } from "@synnaxlabs/x";

import { type Input as BaseInput } from "@/input";
import {
  type ControlStateProps,
  type LabeledConfig,
} from "@/schematic/node/common/symbol/factories";
import { type Setpoint as BaseSetpoint } from "@/vis/setpoint";

export interface Config
  extends
    LabeledConfig,
    Pick<BaseInput.NumericProps, "size">,
    Omit<BaseSetpoint.UseProps, "aetherKey"> {
  dimensions?: dimensions.Dimensions;
  color?: color.Crude;
  units?: string;
  disabled?: boolean;
  control?: ControlStateProps;
}
