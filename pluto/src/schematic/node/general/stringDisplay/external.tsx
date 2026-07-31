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
import { type Config, VARIANT } from "@/schematic/node/general/stringDisplay/config";
import { StringDisplayForm } from "@/schematic/node/general/stringDisplay/Form";
import { StringDisplay } from "@/schematic/node/general/stringDisplay/Primitive";
import { Symbol } from "@/schematic/node/general/stringDisplay/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { Text } from "@/text";
import { type Theming } from "@/theming";

export * from "@/schematic/node/general/stringDisplay/config";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  orientation: "left",
  color: color.ZERO,
  level: "h5",
  inlineSize: 100,
  label: Label.defaultConfig("String Display"),
  stalenessTimeout: 5,
  stalenessColor: t.colors.warning.m1,
  telem: telem.streamChannelStringValue({ channel: 0 }),
});

const Preview = ({ color }: Config): ReactElement => (
  <StringDisplay color={color}>
    <Text.Text>Hello World!</Text.Text>
  </StringDisplay>
);

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "String Display",
  Form: StringDisplayForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
  needsPosition: true,
};
