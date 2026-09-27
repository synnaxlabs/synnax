// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { ValueForm } from "@/schematic/node/general/value/Form";
import { Value } from "@/schematic/node/general/value/Primitive";
import { Symbol } from "@/schematic/node/general/value/Symbol";
import { type Spec } from "@/schematic/node/spec";

const PREVIEW_DIMENSIONS = { width: 60, height: 25 };

const Preview = ({
  color,
  orientation,
  units,
  inlineSize,
}: schematic.ValueNodeConfig): ReactElement => (
  <Value
    color={color}
    orientation={orientation}
    units={units}
    inlineSize={inlineSize}
    dimensions={PREVIEW_DIMENSIONS}
  >
    <Text.Text>50.00</Text.Text>
  </Value>
);

export const spec: Spec<"value", schematic.ValueNodeConfig> = {
  key: "value",
  name: "Value",
  Form: ValueForm,
  Node: Symbol,
  Preview,
  zIndex: 4,
  needsPosition: true,
};
