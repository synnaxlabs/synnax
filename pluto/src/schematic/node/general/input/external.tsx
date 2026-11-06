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
import { SIZE_LEVELS } from "@/schematic/node/common/size";
import { colorFallbacks, InputForm } from "@/schematic/node/general/input/Form";
import { Input } from "@/schematic/node/general/input/Primitive";
import { Symbol } from "@/schematic/node/general/input/Symbol";
import { type Spec } from "@/schematic/node/spec";

const Preview = ({
  fillColor,
  orientation,
  size,
  level,
}: schematic.InputNodeConfig): ReactElement => (
  <Input
    value="send message"
    fillColor={fillColor}
    orientation={orientation}
    size={size}
    level={level}
    disabled
    className={CSS.BM("input-symbol", "preview")}
  />
);

export const spec: Spec<"input", schematic.InputNodeConfig> = {
  key: "input",
  name: "Text input",
  defaults: { level: SIZE_LEVELS.medium },
  Form: InputForm,
  Node: Symbol,
  Preview,
  zIndex: 4,
  colorFallbacks,
  needsPosition: true,
};
