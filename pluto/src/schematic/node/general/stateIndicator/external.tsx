// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Label } from "@/schematic/node/common/label";
import { type Config, VARIANT } from "@/schematic/node/general/stateIndicator/config";
import { StateIndicatorForm } from "@/schematic/node/general/stateIndicator/Form";
import { StateIndicator } from "@/schematic/node/general/stateIndicator/Primitive";
import { Symbol } from "@/schematic/node/general/stateIndicator/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";

export * from "@/schematic/node/general/stateIndicator/config";

export const defaultConfig = (): Config => ({
  variant: VARIANT,
  orientation: "left",
  color: color.ZERO,
  inlineSize: 100,
  options: [],
  label: Label.defaultConfig("State Indicator"),
  source: telem.sourcePipeline("number", {
    connections: [],
    segments: { valueStream: telem.streamChannelValue({ channel: 0 }) },
    outlet: "valueStream",
  }),
});

const Preview = ({ color }: Config): ReactElement => (
  <StateIndicator
    matchedOptionKey="1"
    options={[{ key: "1", name: "Active", value: 1 }]}
    color={color}
  />
);

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "State Indicator",
  Form: StateIndicatorForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
