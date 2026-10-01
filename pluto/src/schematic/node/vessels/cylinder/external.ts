// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { type Spec } from "@/schematic/node/spec";
import { CylinderForm } from "@/schematic/node/vessels/cylinder/Form";
import { Cylinder } from "@/schematic/node/vessels/cylinder/Primitive";
import { Symbol } from "@/schematic/node/vessels/cylinder/Symbol";

export const spec: Spec<"cylinder", schematic.CylinderNodeConfig> = {
  key: "cylinder",
  name: "Cylinder",
  Form: CylinderForm,
  Node: Symbol,
  Preview: Cylinder,
  zIndex: 2,
};
