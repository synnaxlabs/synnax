// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { Form } from "@/schematic/node/common/form";
import { Toggle } from "@/schematic/node/common/toggle";
import { Primitive } from "@/schematic/node/general/switch/primitive";
import type { Spec } from "@/schematic/node/spec";

export const spec: Spec<"switch", schematic.SwitchNodeConfig> = {
  key: "switch",
  name: "Switch",
  Form: (props) => <Form.ToggleForm {...props} hideInnerOrientation />,
  Node: Toggle.createToggle<schematic.SwitchNodeConfig>(Primitive.Switch),
  Preview: Primitive.Switch,
  zIndex: 4,
};

export const SYMBOL = Primitive.SYMBOL;
