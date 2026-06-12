// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box } from "@/schematic/node/general/box";
import { Button } from "@/schematic/node/general/button";
import { Circle } from "@/schematic/node/general/circle";
import { Gauge } from "@/schematic/node/general/gauge";
import { Input } from "@/schematic/node/general/input";
import { Light } from "@/schematic/node/general/light";
import { OffPageReference } from "@/schematic/node/general/offPageReference";
import { Polygon } from "@/schematic/node/general/polygon";
import { Select } from "@/schematic/node/general/select";
import { Setpoint } from "@/schematic/node/general/setpoint";
import { StateIndicator } from "@/schematic/node/general/stateIndicator";
import { Switch } from "@/schematic/node/general/switch";
import { TextBox } from "@/schematic/node/general/textBox";
import { Value } from "@/schematic/node/general/value";

export const REGISTRY = {
  box: Box.spec,
  button: Button.spec,
  circle: Circle.spec,
  gauge: Gauge.spec,
  input: Input.spec,
  light: Light.spec,
  off_page_reference: OffPageReference.spec,
  polygon: Polygon.spec,
  select: Select.spec,
  setpoint: Setpoint.spec,
  state_indicator: StateIndicator.spec,
  switch: Switch.spec,
  text_box: TextBox.spec,
  value: Value.spec,
} as const;
