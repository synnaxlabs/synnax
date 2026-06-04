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
import { type Config, VARIANT } from "@/schematic/node/general/gauge/config";
import { GaugeForm } from "@/schematic/node/general/gauge/Form";
import { Gauge } from "@/schematic/node/general/gauge/Primitive";
import { Symbol } from "@/schematic/node/general/gauge/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";

export * from "@/schematic/node/general/gauge/config";

export const defaultConfig = (): Config => ({
  variant: VARIANT,
  orientation: "left",
  color: color.ZERO,
  units: "RPM",
  level: "h5",
  bounds: bounds.construct(0, 100),
  barWidth: 10,
  label: Label.defaultConfig("Gauge"),
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
  name: "Gauge",
  Form: GaugeForm,
  Node: Symbol,
  Preview: Gauge,
  defaultConfig,
  zIndex: 4,
  needsPosition: true,
};
