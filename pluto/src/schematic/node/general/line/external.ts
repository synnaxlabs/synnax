// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, xy } from "@synnaxlabs/x";

import { Component } from "@/component";
import { type Config, VARIANT } from "@/schematic/node/general/line/config";
import { LineForm } from "@/schematic/node/general/line/Form";
import { Line } from "@/schematic/node/general/line/Primitive";
import { Symbol } from "@/schematic/node/general/line/Symbol";
import { type Spec } from "@/schematic/node/spec";

export * from "@/schematic/node/general/line/config";

export const defaultConfig = (): Config => ({
  variant: VARIANT,
  color: color.ZERO,
  start: xy.ZERO,
  end: { x: 100, y: 0 },
  strokeWidth: 2,
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Line",
  Form: LineForm,
  Node: Symbol,
  Preview: Component.removeProps(Line, ["start", "end"]),
  defaultConfig,
  zIndex: 2,
  needsPosition: true,
};
