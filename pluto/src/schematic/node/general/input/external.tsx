// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Label } from "@/schematic/node/common/label";
import { type Config } from "@/schematic/node/general/input/config";
import { InputForm } from "@/schematic/node/general/input/Form";
import { Primitive } from "@/schematic/node/general/input/Primitive";
import { Symbol } from "@/schematic/node/general/input/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Theming } from "@/theming";

export type { Config };

export const VARIANT = "input";
export const NAME = "Input";

export const defaultConfig = (t: Theming.Theme): Config => ({
  orientation: "left",
  color: t.colors.gray.l11,
  size: "small",
  label: Label.defaultConfig("Input"),
  sink: telem.sinkPipeline("string", {
    connections: [],
    segments: { setter: control.setChannelValue({ channel: 0 }) },
    inlet: "setter",
  }),
});

const Preview = ({ color }: Config): ReactElement => (
  <Primitive
    value="send message"
    onChange={() => {}}
    color={color}
    disabled
    className={CSS.BM("input-symbol", "preview")}
  />
);

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: NAME,
  Form: InputForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
