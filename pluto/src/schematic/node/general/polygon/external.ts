// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { colorFallbacks, PolygonForm } from "@/schematic/node/general/polygon/Form";
import { Preview, Symbol } from "@/schematic/node/general/polygon/Symbol";
import { type Spec } from "@/schematic/node/spec";

export const spec: Spec<"polygon", schematic.PolygonNodeConfig> = {
  key: "polygon",
  name: "Polygon",
  Form: PolygonForm,
  Node: Symbol,
  Preview,
  zIndex: 2,
  colorFallbacks,
};
