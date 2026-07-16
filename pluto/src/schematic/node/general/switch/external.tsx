// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import { Switch } from "@/schematic/node/general/switch/Primitive";
import { type Spec } from "@/schematic/node/spec";

export const defaultConfig = (): schematic.NodeConfigSwitch => ({
  variant: "switch",
  label: Label.defaultConfig("Switch"),
  ...Primitive.ZERO_PROPS,
  ...Toggle.ZERO_TOGGLE_DEFAULTS,
});

export const spec: Spec<"switch", schematic.NodeConfigSwitch> = {
  key: "switch",
  name: "Switch",
  Form: () => <Form.ToggleForm hideInnerOrientation omit={["onClickDelay"]} />,
  Node: Toggle.createToggle<schematic.NodeConfigSwitch>(Switch),
  Preview: Switch,
  defaultConfig,
  zIndex: 4,
};
