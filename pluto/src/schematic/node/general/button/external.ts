// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { ButtonForm } from "@/schematic/node/general/button/Form";
import { Button } from "@/schematic/node/general/button/Primitive";
import { Symbol } from "@/schematic/node/general/button/Symbol";
import { type Spec } from "@/schematic/node/spec";

const NAME = "Button";

export const spec: Spec<"button", schematic.ButtonNodeConfig> = {
  key: "button",
  name: NAME,
  Form: ButtonForm,
  Node: Symbol,
  Preview: Button,
  zIndex: 4,
};
