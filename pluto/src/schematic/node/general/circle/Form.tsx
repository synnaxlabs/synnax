// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/charon/flex";
import type { Input } from "@synnaxlabs/charon/input";
import { type ReactElement } from "react";

import { Form as Base } from "@synnaxlabs/charon/form";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
const RADIUS_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.5, y: 2.5 },
  bounds: { lower: 0, upper: 500 },
  endContent: "px",
};

export const CircleForm = (): ReactElement => (
  <Form.Wrapper direction="x" align="stretch">
    <Flex.Box direction="y" grow>
      <Label.Form path="label" />
      <Flex.Box direction="x">
        <Form.ColorField path="color" />
        <Form.ColorField path="backgroundColor" label="Background Color" />
        <Base.NumericField
          path="radius"
          label="Radius"
          inputProps={RADIUS_INPUT_PROPS}
          grow
        />
        <Base.NumericField
          path="strokeWidth"
          label="Border Width"
          inputProps={Form.STROKE_WIDTH_INPUT_PROPS}
          grow
        />
      </Flex.Box>
    </Flex.Box>
  </Form.Wrapper>
);
