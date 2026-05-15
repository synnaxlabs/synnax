// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Text } from "@synnaxlabs/lyra/text";
import type { Theming } from "@synnaxlabs/lyra/theming";
import { type ReactElement } from "react";

import { Label } from "@/schematic/node/common/label";
import { type Config, VARIANT } from "@/schematic/node/general/value/config";
import { ValueForm } from "@/schematic/node/general/value/Form";
import { Primitive } from "@/schematic/node/general/value/Primitive";
import { Symbol } from "@/schematic/node/general/value/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { Value as BaseValue } from "@/vis/value";

export * from "@/schematic/node/general/value/config";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  orientation: "left",
  color: t.colors.gray.l11,
  units: "psi",
  level: "h5",
  inlineSize: 70,
  label: Label.defaultConfig("Value"),
  stalenessTimeout: 5,
  stalenessColor: t.colors.warning.m1,
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
  redline: BaseValue.ZERO_READLINE,
});

const PREVIEW_DIMENSIONS = { width: 60, height: 25 };

const Preview = ({ color }: Config): ReactElement => (
  <Primitive color={color} dimensions={PREVIEW_DIMENSIONS} units="psi">
    <Text.Text>50.00</Text.Text>
  </Primitive>
);

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Value",
  Form: ValueForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
  needsPosition: true,
};
