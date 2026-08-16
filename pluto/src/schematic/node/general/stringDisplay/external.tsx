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
import { StringDisplayForm } from "@/schematic/node/general/stringDisplay/Form";
import { StringDisplay } from "@/schematic/node/general/stringDisplay/Primitive";
import { Symbol } from "@/schematic/node/general/stringDisplay/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export const defaultConfig = (t: Theming.Theme): schematic.StringDisplayNodeConfig => ({
  variant: "string_display",
  orientation: "left",
  color: color.ZERO,
  level: "p",
  inlineSize: 100,
  label: Label.defaultConfig("String Display"),
  stalenessTimeout: 5,
  stalenessColor: t.colors.warning.m1,
});

const Preview = ({ color, level }: schematic.StringDisplayNodeConfig): ReactElement => (
  <StringDisplay color={color} level={level} value="Hello World!" />
);

export const spec: Spec<"string_display", schematic.StringDisplayNodeConfig> = {
  key: "string_display",
  name: "String Display",
  Form: StringDisplayForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
