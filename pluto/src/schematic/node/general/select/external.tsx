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

import { CSS } from "@/css";
import { Label } from "@/schematic/node/common/label";
import { type Config, VARIANT } from "@/schematic/node/general/select/config";
import { SelectForm } from "@/schematic/node/general/select/Form";
import { Select } from "@/schematic/node/general/select/Primitive";
import { Symbol } from "@/schematic/node/general/select/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";

export * from "@/schematic/node/general/select/config";

export const defaultConfig = (): Config => ({
  variant: VARIANT,
  orientation: "left",
  color: color.ZERO,
  size: "small",
  inlineSize: 100,
  options: [],
  label: Label.defaultConfig("Select"),
  control: { show: true },
  sink: telem.sinkPipeline("number", {
    connections: [],
    segments: { setter: control.setChannelValue({ channel: 0 }) },
    inlet: "setter",
  }),
});

const Preview = ({ color }: Config): ReactElement => (
  <Select
    onChange={() => {}}
    options={[]}
    color={color}
    disabled
    className={CSS.BM("select-symbol", "preview")}
  />
);

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Select",
  Form: SelectForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
