// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { GaugeForm } from "@/schematic/node/general/gauge/Form";
import { Gauge } from "@/schematic/node/general/gauge/Primitive";
import { Symbol } from "@/schematic/node/general/gauge/Symbol";
import { type Spec } from "@/schematic/node/spec";

export const spec: Spec<"gauge", schematic.GaugeNodeConfig> = {
  key: "gauge",
  name: "Gauge",
  Form: GaugeForm,
  Node: Symbol,
  Preview: Gauge,
  zIndex: 4,
  needsPosition: true,
};
