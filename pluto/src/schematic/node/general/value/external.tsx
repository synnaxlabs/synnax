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

import { ValueForm } from "@/schematic/node/general/value/Form";
import { Value } from "@/schematic/node/general/value/Primitive";
import { Symbol } from "@/schematic/node/general/value/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { Text } from "@/text";

const PREVIEW_HEIGHT = 25;
const PREVIEW_INLINE_SIZE = 60;

// The picker draws its own text rather than the canvas the placed symbol uses, so it
// takes the code typeface to read as the same symbol.
const Preview = ({
  color,
  orientation,
  units,
}: schematic.ValueNodeConfig): ReactElement => (
  <Value
    color={color}
    orientation={orientation}
    height={PREVIEW_HEIGHT}
    inlineSize={PREVIEW_INLINE_SIZE}
    units={units}
  >
    <Text.Text variant="code">50.00</Text.Text>
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
