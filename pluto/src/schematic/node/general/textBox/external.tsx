// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Label } from "@/schematic/node/common/label";
import { TextBoxForm } from "@/schematic/node/general/textBox/Form";
import { TextBox } from "@/schematic/node/general/textBox/Primitive";
import { Symbol } from "@/schematic/node/general/textBox/Symbol";
import { type Spec } from "@/schematic/node/spec";

export const defaultConfig = (): schematic.NodeConfigTextBox => ({
  variant: "text_box",
  orientation: "left",
  color: color.ZERO,
  autoFit: true,
  align: "center",
  label: Label.defaultConfig("Text Box"),
  level: "p",
  value: "Text Box",
  width: 75,
});

const Preview = (props: schematic.NodeConfigTextBox): ReactElement => (
  <TextBox {...props} autoFit value="Text Box" />
);

export const spec: Spec<"text_box", schematic.NodeConfigTextBox> = {
  key: "text_box",
  name: "Text Box",
  Form: TextBoxForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
