// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
import { type schematic } from "@synnaxlabs/client";

import { Component } from "@/component";
import { LineForm } from "@/schematic/node/general/line/Form";
import { Line } from "@/schematic/node/general/line/Primitive";
import { Symbol } from "@/schematic/node/general/line/Symbol";
import { type Spec } from "@/schematic/node/spec";

export const spec: Spec<"line", schematic.LineNodeConfig> = {
  key: "line",
  name: "Line",
  Form: LineForm,
  Node: Symbol,
  Preview: Component.removeProps(Line, ["start", "end"]),
  zIndex: 2,
  needsPosition: true,
};
