// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { Label } from "@/schematic/node/common/label";
import { ButtonForm } from "@/schematic/node/general/button/Form";
import { Button } from "@/schematic/node/general/button/Primitive";
import { Symbol } from "@/schematic/node/general/button/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

const NAME = "Button";

export const defaultConfig = (t: Theming.Theme): schematic.NodeConfigButton => ({
  variant: "button",
  orientation: "left",
  color: t.colors.primary.z,
  label: Label.defaultConfig(NAME),
  control: { show: true },
  mode: "fire",
  onClickDelay: 0,
});

export const spec: Spec<"button", schematic.NodeConfigButton> = {
  key: "button",
  name: NAME,
  Form: ButtonForm,
  Node: Symbol,
  Preview: Button,
  defaultConfig,
  zIndex: 4,
};
