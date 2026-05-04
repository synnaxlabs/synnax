// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds } from "@synnaxlabs/x";

import { Label } from "@/schematic/node/common/label";
import { type Config } from "@/schematic/node/general/gauge/config";
import { GaugeForm } from "@/schematic/node/general/gauge/Form";
import { Primitive } from "@/schematic/node/general/gauge/Primitive";
import { Symbol } from "@/schematic/node/general/gauge/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { type Theming } from "@/theming";

export type { Config };

export const VARIANT = "gauge";
export const NAME = "Gauge";

export const defaultConfig = (t: Theming.Theme): Config => ({
  orientation: "left",
  color: t.colors.gray.l11,
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
  name: NAME,
  Form: GaugeForm,
  Node: Symbol,
  Preview: Primitive,
  defaultConfig,
  zIndex: 4,
  needsPosition: true,
};
