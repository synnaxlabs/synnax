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
import { type ReactElement } from "react";

import { Label } from "@/schematic/node/common/label";
import { StateIndicatorForm } from "@/schematic/node/general/stateIndicator/Form";
import { StateIndicator } from "@/schematic/node/general/stateIndicator/Primitive";
import { Symbol } from "@/schematic/node/general/stateIndicator/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export const defaultConfig = (
  t: Theming.Theme,
): schematic.StateIndicatorNodeConfig => ({
  variant: "state_indicator",
  orientation: "left",
  color: color.ZERO,
  inlineSize: 100,
  options: [],
  label: Label.defaultConfig("State Indicator"),
  stalenessTimeout: 5,
  stalenessColor: t.colors.warning.m1,
});

const Preview = ({ color }: schematic.StateIndicatorNodeConfig): ReactElement => (
  <StateIndicator
    matchedOptionKey="1"
    options={[{ key: "1", name: "Active", value: 1 }]}
    color={color}
  />
);

export const spec: Spec<"state_indicator", schematic.StateIndicatorNodeConfig> = {
  key: "state_indicator",
  name: "State Indicator",
  Form: StateIndicatorForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
