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
import { type Spec } from "@/schematic/node/spec";
import { TJunction } from "@/schematic/node/vessels/tJunction/Primitive";

export const spec: Spec<"t_junction", schematic.TJunctionNodeConfig> = {
  key: "t_junction",
  name: "T Junction",
  label: "",
  Form: Form.StyleForm,
  Node: Label.createLabeled<schematic.TJunctionNodeConfig>(TJunction),
  Preview: TJunction,
  zIndex: 24,
};
