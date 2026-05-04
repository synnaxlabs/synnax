// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CommonToggleForm } from "@/schematic/node/common/forms";
import { Label } from "@/schematic/node/common/label";
import { createToggle } from "@/schematic/node/common/symbol/factories";
import { type Config } from "@/schematic/node/pumps/rollerVane/config";
import { Primitive } from "@/schematic/node/pumps/rollerVane/Primitive";
import { type Spec } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Theming } from "@/theming";
export type { Config };

export const VARIANT = "rollerVaneCompressor";
export const NAME = "Roller Vane Compressor";
const ZERO_PROPS = { orientation: "left" as const, scale: 1 };

const ZERO_BOOLEAN_SOURCE_PROPS = {
  ...ZERO_PROPS,
  source: telem.sourcePipeline("boolean", {
    connections: [{ from: "valueStream", to: "threshold" }],
    segments: {
      valueStream: telem.streamChannelValue({ channel: 0 }),
      threshold: telem.withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
    },
    outlet: "threshold",
  }),
};

const ZERO_BOOLEAN_SINK_PROPS = {
  ...ZERO_PROPS,
  control: { show: true },
  sink: telem.sinkPipeline("boolean", {
    connections: [{ from: "setpoint", to: "setter" }],
    segments: {
      setter: control.setChannelValue({ channel: 0 }),
      setpoint: telem.setpoint({ truthy: 1, falsy: 0 }),
    },
    inlet: "setpoint",
  }),
};

const ZERO_TOGGLE_PROPS = {
  ...ZERO_BOOLEAN_SOURCE_PROPS,
  ...ZERO_BOOLEAN_SINK_PROPS,
  onClickDelay: 0,
};
export const defaultConfig = (t: Theming.Theme): Config => ({
  color: t.colors.gray.l11,
  label: Label.defaultConfig("Roller Vane Compressor"),
  ...ZERO_TOGGLE_PROPS,
});
export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: NAME,
  Form: CommonToggleForm,
  Node: createToggle<Config>(Primitive),
  Preview: Primitive,
  defaultConfig,
  zIndex: 4,
};
