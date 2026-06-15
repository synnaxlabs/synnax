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

import { CSS } from "@/css";
import { Label } from "@/schematic/node/common/label";
import { InputForm } from "@/schematic/node/general/input/Form";
import { Input } from "@/schematic/node/general/input/Primitive";
import { Symbol } from "@/schematic/node/general/input/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export const defaultConfig = (t: Theming.Theme): schematic.NodeConfigInput => ({
  variant: "input",
  orientation: "left",
  color: t.colors.gray.l11,
  size: "small",
  label: Label.defaultConfig("Input"),
  control: { show: true },
});

const Preview = ({ color }: schematic.NodeConfigInput): ReactElement => (
  <Input
    initialValue="send message"
    color={color}
    disabled
    className={CSS.BM("input-symbol", "preview")}
  />
);

export const spec: Spec<"input", schematic.NodeConfigInput> = {
  key: "input",
  name: "Input",
  Form: InputForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
