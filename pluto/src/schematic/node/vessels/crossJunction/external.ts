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
import { CrossJunction } from "@/schematic/node/vessels/crossJunction/Primitive";

export const spec: Spec<"cross_junction", schematic.CrossJunctionNodeConfig> = {
  key: "cross_junction",
  name: "Cross junction",
  label: "",
  Form: Form.StyleForm,
  Node: Label.createLabeled<schematic.CrossJunctionNodeConfig>(CrossJunction),
  Preview: CrossJunction,
  zIndex: 24,
};
