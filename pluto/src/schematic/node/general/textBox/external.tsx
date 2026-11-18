// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Color } from "@/color";
import { TextBoxForm } from "@/schematic/node/general/textBox/Form";
import { TextBox } from "@/schematic/node/general/textBox/Primitive";
import { Symbol } from "@/schematic/node/general/textBox/Symbol";
import { type Spec } from "@/schematic/node/spec";

const Preview = ({
  label: _label,
  textColor,
  ...rest
}: schematic.TextBoxNodeConfig): ReactElement => {
  const resolve = Color.useResolve();
  return <TextBox {...rest} textColor={resolve(textColor)} value="Text box" />;
};

export const spec: Spec<"text_box", schematic.TextBoxNodeConfig> = {
  key: "text_box",
  name: "Text box",
  value: "Text box",
  Form: TextBoxForm,
  Node: Symbol,
  Preview,
  zIndex: 4,
};
