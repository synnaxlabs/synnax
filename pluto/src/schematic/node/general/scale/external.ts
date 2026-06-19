// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, color } from "@synnaxlabs/x";

import { Label } from "@/schematic/node/common/label";
import { type Config, VARIANT } from "@/schematic/node/general/scale/config";
import { ScaleForm } from "@/schematic/node/general/scale/Form";
import { Scale } from "@/schematic/node/general/scale/Primitive";
import { Symbol } from "@/schematic/node/general/scale/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";

export * from "@/schematic/node/general/scale/config";

export const defaultConfig = (): Config => ({
  variant: VARIANT,
  orientation: "left",
  color: color.ZERO,
  bounds: bounds.construct(0, 100),
  style: "fill",
  side: "right",
  level: "small",
  dimensions: { width: 60, height: 160 },
  label: Label.defaultConfig("Scale"),
  telem: telem.sourcePipeline("string", {
    connections: [
      { from: "valueStream", to: "rollingAverage" },
      { from: "rollingAverage", to: "stringifier" },
    ],
    segments: {
      valueStream: telem.streamChannelValue({ channel: 0 }),
      rollingAverage: telem.rollingAverage({ windowSize: 1 }),
      stringifier: telem.stringifyNumber({ precision: 2, notation: "standard" }),
    },
    outlet: "stringifier",
  }),
});

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Scale",
  Form: ScaleForm,
  Node: Symbol,
  Preview: Scale,
  defaultConfig,
  zIndex: 4,
  needsPosition: true,
};
