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
import { SelectForm } from "@/schematic/node/general/select/Form";
import { Select } from "@/schematic/node/general/select/Primitive";
import { Symbol } from "@/schematic/node/general/select/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export const defaultConfig = (t: Theming.Theme): schematic.NodeConfigSelect => ({
  variant: "select",
  orientation: "left",
  color: t.colors.gray.l11,
  size: "small",
  inlineSize: 100,
  options: [],
  label: Label.defaultConfig("Select"),
  control: { show: true },
});

const Preview = ({ color }: schematic.NodeConfigSelect): ReactElement => (
  <Select
    onChange={() => {}}
    options={[]}
    color={color}
    disabled
    className={CSS.BM("select-symbol", "preview")}
  />
);

export const spec: Spec<"select", schematic.NodeConfigSelect> = {
  key: "select",
  name: "Select",
  Form: SelectForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
