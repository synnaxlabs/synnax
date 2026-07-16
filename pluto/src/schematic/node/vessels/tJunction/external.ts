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

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Spec } from "@/schematic/node/spec";
import { TJunction } from "@/schematic/node/vessels/tJunction/Primitive";

export const defaultConfig = (): schematic.NodeConfigTJunction => ({
  variant: "t_junction",
  color: color.ZERO,
  label: Label.defaultConfig(""),
  ...Primitive.ZERO_PROPS,
});

export const spec: Spec<"t_junction", schematic.NodeConfigTJunction> = {
  key: "t_junction",
  name: "T Junction",
  Form: Form.StyleForm,
  Node: Label.createLabeled<schematic.NodeConfigTJunction>(TJunction),
  Preview: TJunction,
  defaultConfig,
  zIndex: 24,
};
