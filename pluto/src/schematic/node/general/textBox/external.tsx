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
import { type Config, VARIANT } from "@/schematic/node/general/textBox/config";
import { TextBoxForm } from "@/schematic/node/general/textBox/Form";
import { TextBox } from "@/schematic/node/general/textBox/Primitive";
import { Symbol } from "@/schematic/node/general/textBox/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export * from "@/schematic/node/general/textBox/config";

export const defaultConfig = (_t: Theming.Theme): Config => ({
  variant: VARIANT,
  orientation: "left",
  color: color.ZERO,
  autoFit: true,
  align: "center",
  label: Label.defaultConfig("Text Box"),
  level: "p",
  value: "Text Box",
  width: 75,
});

const Preview = (props: Config): ReactElement => (
  <TextBox {...props} autoFit value="Text Box" />
);

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Text Box",
  Form: TextBoxForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
